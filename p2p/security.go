package p2p

import (
	"errors"
	"net"
	"strings"
	"sync"
	"time"
)

// === Limitler & varsayılanlar ===

// MaxMessageSize documents the QuantumCoin component used by this module.
// Tek bir P2P mesajının izin verilen maksimum boyutu (byte)
const MaxMessageSize = 1 << 20 // 1 MiB

// MaxConnectionsPerIP documents the QuantumCoin component used by this module.
// Aynı IP'den aynı anda en fazla kaç aktif bağlantı?
const MaxConnectionsPerIP = 32

// RateWindow documents the QuantumCoin component used by this module.
// Mesaj rate-limit penceresi
const RateWindow = 10 * time.Second

// MaxMessagesPerIPInWindow documents the QuantumCoin component used by this module.
// RateWindow içinde bir IP'den en fazla kaç mesaja izin var?
const MaxMessagesPerIPInWindow = 200

// BanDurationDefault documents the QuantumCoin component used by this module.
// Otomatik ban süresi
const BanDurationDefault = 30 * time.Minute

// Hata tipleri
var (
	ErrBannedPeer   = errors.New("p2p: peer is banned")
	ErrTooManyConns = errors.New("p2p: too many connections from this IP")
	ErrMsgTooLarge  = errors.New("p2p: message too large")
	ErrRateLimited  = errors.New("p2p: message rate limited")
)

// PeerSecurity documents the QuantumCoin component used by this module.
// PeerSecurity: IP bazlı ban, rate-limit ve reputation yönetimi
type PeerSecurity struct {
	mu sync.Mutex

	// IP -> ban bitiş zamanı
	bannedUntil map[string]time.Time

	// IP -> son mesaj zamanları (sadece son RateWindow içindekiler)
	msgTimestamps map[string][]time.Time

	// IP -> aktif bağlantı sayısı
	connCount map[string]int

	// PEER_REPUTATION_ENGINE_V1
	// IP -> reputation score (0-100)
	reputation map[string]int
}

// DefaultPeerSecurity documents the QuantumCoin component used by this module.
// Global security objesi – tüm p2p bu objeyi kullanacak
var DefaultPeerSecurity = NewPeerSecurity()

// NewPeerSecurity documents the QuantumCoin component used by this module.
// Yeni bir security yöneticisi oluştur
func NewPeerSecurity() *PeerSecurity {
	return &PeerSecurity{
		bannedUntil:   make(map[string]time.Time),
		msgTimestamps: make(map[string][]time.Time),
		connCount:     make(map[string]int),
		reputation:    make(map[string]int),
	}
}

// ExtractIP documents the QuantumCoin component used by this module.
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

// IsBanned documents the QuantumCoin component used by this module.
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

// Ban documents the QuantumCoin component used by this module.
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
	ps.adjustReputationLocked(ip, -30)
}

// OnConnect documents the QuantumCoin component used by this module.
// Yeni bağlantı açılırken çağrılır
//   - ban kontrolü
//   - aynı IP'den aktif connection sayısı
func (ps *PeerSecurity) OnConnect(ip string) error {
	if ip == "" {
		return nil
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.ensureReputationLocked(ip)

	// Ban kontrolü
	if until, ok := ps.bannedUntil[ip]; ok {
		if time.Now().Before(until) {
			ps.adjustReputationLocked(ip, -5)
			return ErrBannedPeer
		}
		// Ban süresi bittiyse sil
		delete(ps.bannedUntil, ip)
	}

	// Aktif bağlantı sayısı
	n := ps.connCount[ip]
	if n >= MaxConnectionsPerIP {
		ps.adjustReputationLocked(ip, -10)
		return ErrTooManyConns
	}
	ps.connCount[ip] = n + 1
	ps.adjustReputationLocked(ip, 1)
	return nil
}

// OnDisconnect documents the QuantumCoin component used by this module.
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

// AllowMessage documents the QuantumCoin component used by this module.
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

	ps.ensureReputationLocked(ip)

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
		ps.adjustReputationLocked(ip, -25)
		return ErrRateLimited
	}

	ps.adjustReputationLocked(ip, 1)
	return nil
}

// ClearStaleData documents the QuantumCoin component used by this module.
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

// PEER_REPUTATION_ENGINE_V1
func (ps *PeerSecurity) ensureReputationLocked(ip string) int {
	if ip == "" {
		return 0
	}
	if ps.reputation == nil {
		ps.reputation = make(map[string]int)
	}
	score, ok := ps.reputation[ip]
	if !ok {
		score = 100
		ps.reputation[ip] = score
	}
	return score
}

func (ps *PeerSecurity) adjustReputationLocked(ip string, delta int) int {
	score := ps.ensureReputationLocked(ip)
	score += delta
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	ps.reputation[ip] = score
	return score
}

func (ps *PeerSecurity) AdjustReputation(ip string, delta int) int {
	if ip == "" {
		return 0
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.adjustReputationLocked(ip, delta)
}

func (ps *PeerSecurity) Reputation(ip string) int {
	if ip == "" {
		return 0
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.ensureReputationLocked(ip)
}

func (ps *PeerSecurity) ReputationStatus(ip string) string {
	score := ps.Reputation(ip)
	switch {
	case score >= 80:
		return "TRUSTED"
	case score >= 50:
		return "WATCH"
	case score >= 20:
		return "RISKY"
	default:
		return "QUARANTINE"
	}
}

func (ps *PeerSecurity) ReputationSnapshot() map[string]any {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.reputation == nil {
		ps.reputation = make(map[string]int)
	}

	items := map[string]any{}
	for ip, score := range ps.reputation {
		status := "TRUSTED"
		switch {
		case score >= 80:
			status = "TRUSTED"
		case score >= 50:
			status = "WATCH"
		case score >= 20:
			status = "RISKY"
		default:
			status = "QUARANTINE"
		}
		items[ip] = map[string]any{
			"score":  score,
			"status": status,
		}
	}

	return map[string]any{
		"version": "PEER_REPUTATION_ENGINE_V1",
		"peers":   items,
		"safeBoundary": map[string]bool{
			"changesChain":       false,
			"changesBalances":    false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
			"performsTransfer":   false,
			"startsMining":       false,
			"replacesChain":      false,
		},
	}
}
