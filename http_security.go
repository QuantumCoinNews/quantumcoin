package main

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

//
// === GLOBAL HTTP GUARD (IP bazlı rate-limit + body size limiti) ===
//

// Tek bir HTTP request body için maksimum boyut (byte)
const httpMaxBodyBytes int64 = 1 << 20 // 1 MiB

// IP bazlı rate-limit penceresi
const httpRateWindow = 10 * time.Second

// httpRateWindow içinde bir IP'den en fazla kaç request?
const httpMaxRequestsPerIPInWindow = 100

// Otomatik ban süresi
const httpBanDuration = 10 * time.Minute

var (
	ErrHTTPBanned    = errors.New("http guard: banned")
	ErrHTTPRateLimit = errors.New("http guard: too many requests")
)

type httpClientState struct {
	timestamps  []time.Time
	bannedUntil time.Time
}

type HTTPGuard struct {
	mu      sync.Mutex
	clients map[string]*httpClientState
}

var defaultHTTPGuard = NewHTTPGuard()

func NewHTTPGuard() *HTTPGuard {
	return &HTTPGuard{
		clients: make(map[string]*httpClientState),
	}
}

// Basit IP çıkarma (X-Forwarded-For vs. ile uğraşmıyoruz şimdilik)
func clientIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (g *HTTPGuard) allow(ip string) error {
	if ip == "" {
		return nil
	}

	now := time.Now()
	windowStart := now.Add(-httpRateWindow)

	g.mu.Lock()
	defer g.mu.Unlock()

	st, ok := g.clients[ip]
	if !ok {
		st = &httpClientState{}
		g.clients[ip] = st
	}

	// Banlı mı?
	if !st.bannedUntil.IsZero() && now.Before(st.bannedUntil) {
		return ErrHTTPBanned
	}
	if !st.bannedUntil.IsZero() && now.After(st.bannedUntil) {
		// ban süresi bitmişse kaldır
		st.bannedUntil = time.Time{}
	}

	// Eski timestamp'leri temizle
	h := st.timestamps
	j := 0
	for _, ts := range h {
		if ts.After(windowStart) {
			h[j] = ts
			j++
		}
	}
	h = h[:j]

	// Yeni request'i ekle
	h = append(h, now)
	st.timestamps = h

	if len(h) > httpMaxRequestsPerIPInWindow {
		st.bannedUntil = now.Add(httpBanDuration)
		return ErrHTTPRateLimit
	}

	return nil
}

// Tüm HTTP API'yi sarmak için middleware
func (g *HTTPGuard) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIPFromRequest(r)

		if err := g.allow(ip); err != nil {
			switch err {
			case ErrHTTPBanned:
				http.Error(w, "Forbidden", http.StatusForbidden)
			case ErrHTTPRateLimit:
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			default:
				http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			}
			return
		}

		// Body boyut limiti (DoS'a karşı)
		r.Body = http.MaxBytesReader(w, r.Body, httpMaxBodyBytes)

		next.ServeHTTP(w, r)
	})
}

//
// === KRİTİK ENDPOINT LİMİTLEYİCİ (uç bazlı ince ayar) ===
//

type EndpointLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time // key: ip|endpointKey
}

func NewEndpointLimiter() *EndpointLimiter {
	return &EndpointLimiter{
		hits: make(map[string][]time.Time),
	}
}

// Global kritik limiter
var criticalLimiter = NewEndpointLimiter()

// ip + endpoint key bazında pencere/limit kontrolü
func (el *EndpointLimiter) allow(ip, key string, window time.Duration, max int) bool {
	if ip == "" || key == "" || max <= 0 {
		return true
	}

	now := time.Now()
	windowStart := now.Add(-window)
	compoundKey := ip + "|" + key

	el.mu.Lock()
	defer el.mu.Unlock()

	h := el.hits[compoundKey]
	j := 0
	for _, ts := range h {
		if ts.After(windowStart) {
			h[j] = ts
			j++
		}
	}
	h = h[:j]

	h = append(h, now)
	el.hits[compoundKey] = h

	return len(h) <= max
}

// Belirli bir endpoint için HandlerFunc saran yardımcı
func (el *EndpointLimiter) WrapFunc(key string, window time.Duration, max int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIPFromRequest(r)
		if !el.allow(ip, key, window, max) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}
