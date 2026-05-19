package p2p

import (
	"errors"
	"net"
	"strings"
	"sync"
	"time"
)

// === Limitler & varsayılanlar ===

// Tek bir P2P mesajının izin verilen maksimum boyutu (byte)
const MaxMessageSize = 1 << 20 // 1 MiB

// Aynı IP'den aynı anda en fazla kaç aktif bağlantı?
const MaxConnectionsPerIP = 32

// Mesaj rate-limit penceresi
const RateWindow = 10 * time.Second

// RateWindow içinde bir IP'den en fazla kaç mesaja izin var?
const MaxMessagesPerIPInWindow = 200

// Otomatik ban süresi
const BanDurationDefault = 30 * time.Minute

// Hata tipleri
var (
	ErrBannedPeer   = errors.New("p2p: peer is banned")
	ErrTooManyConns = errors.New("p2p: too many connections from this IP")
	ErrMsgTooLarge  = errors.New("p2p: message too large")
	ErrRateLimited  = errors.New("p2p: message rate limited")
)

// PeerSecurity: IP bazlı ban ve rate-limit yönetimi
type PeerSecurity struct {
	mu sync.Mutex

	// IP -> ban bitiş zamanı
	bannedUntil map[string]time.Time

	// IP -> son mesaj zamanları (sadece son RateWindow içindekiler)
	msgTimestamps map[string][]time.Time

	// IP -> aktif bağlantı sayısı
	connCount map[string]int
}

// Global security objesi – tüm p2p bu objeyi kullanacak
var DefaultPeerSecurity = NewPeerSecurity()

// Yeni bir security yöneticisi oluştur
func NewPeerSecurity() *PeerSecurity {
	return &PeerSecurity{
		bannedUntil:   make(map[string]time.Time),
		msgTimestamps: make(map[string][]time.Time),
		connCount:     make(map[string]int),
	}
}

// net.Addr içinden IP kısmını çıkar
func ExtractIP(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host := addr.String() // "ip:port" veya "[ip]:port"

	// IPv6 için köşeli parantezleri temizle
	if strings.HasPrefix(host, "[") {
		if i := strings.LastIndex(host, "]"); i != -1 {
			host = host[1:i]
		}
	} else {
		if i := strings.LastIndex(host, ":"); i != -1 {
			host = host[:i]
		}
	}
	return strings.TrimSpace(host)
}

// IP şu an banlı mı?
func (ps *PeerSecurity) IsBanned(ip string) bool {
	if ip == "" {
		return false
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	until, ok := ps.bannedUntil[ip]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		// Süresi dolmuş ban'ı temizle
		delete(ps.bannedUntil, ip)
		return false
	}
	return true
}

// IP'yi belirli süre için banla
func (ps *PeerSecurity) Ban(ip string, dur time.Duration) {
	if ip == "" {
		return
	}
	if dur <= 0 {
		dur = BanDurationDefault
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.bannedUntil[ip] = time.Now().Add(dur)
}

// Yeni bağlantı açılırken çağrılır
//   - ban kontrolü
//   - aynı IP'den aktif connection sayısı
func (ps *PeerSecurity) OnConnect(ip string) error {
	if ip == "" {
		return nil
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Ban kontrolü
	if until, ok := ps.bannedUntil[ip]; ok {
		if time.Now().Before(until) {
			return ErrBannedPeer
		}
		// Ban süresi bittiyse sil
		delete(ps.bannedUntil, ip)
	}

	// Aktif bağlantı sayısı
	n := ps.connCount[ip]
	if n >= MaxConnectionsPerIP {
		return ErrTooManyConns
	}
	ps.connCount[ip] = n + 1
	return nil
}

// Bağlantı kapandığında çağrılır
func (ps *PeerSecurity) OnDisconnect(ip string) {
	if ip == "" {
		return
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	n := ps.connCount[ip]
	if n <= 1 {
		delete(ps.connCount, ip)
	} else {
		ps.connCount[ip] = n - 1
	}
}

// Mesaj kabul etmeden önce boyut + rate-limit kontrolü
func (ps *PeerSecurity) AllowMessage(ip string, msgLen int) error {
	if ip == "" {
		return nil
	}

	// Çok büyük mesaj → direkt banla
	if msgLen < 0 || msgLen > MaxMessageSize {
		ps.Ban(ip, BanDurationDefault)
		return ErrMsgTooLarge
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-RateWindow)

	// Eski timestamp'leri temizle
	history := ps.msgTimestamps[ip]
	j := 0
	for _, ts := range history {
		if ts.After(windowStart) {
			history[j] = ts
			j++
		}
	}
	history = history[:j]

	// Yeni mesajı ekle
	history = append(history, now)
	ps.msgTimestamps[ip] = history

	// Flood tespiti
	if len(history) > MaxMessagesPerIPInWindow {
		ps.bannedUntil[ip] = now.Add(BanDurationDefault)
		return ErrRateLimited
	}

	return nil
}

// İsteğe bağlı temizlik (çok eski IP kayıtlarını silmek için)
func (ps *PeerSecurity) ClearStaleData(maxIdle time.Duration) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	cut := time.Now().Add(-maxIdle)
	for ip, history := range ps.msgTimestamps {
		if len(history) == 0 {
			delete(ps.msgTimestamps, ip)
			continue
		}
		last := history[len(history)-1]
		if last.Before(cut) {
			delete(ps.msgTimestamps, ip)
		}
	}
}
