package main

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"quantumcoin/security"
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

const (
	httpLimiterShardCount                 = 32
	httpGuardMaxKeysPerShard              = 256
	endpointLimiterMaxKeysPerShard        = 256
	httpLimiterSweepEvery          uint64 = 64
	httpSecurityEventCooldown             = time.Second
	httpGuardStateRetention               = httpBanDuration + httpRateWindow
	endpointLimiterMinRetention           = time.Minute
)

var (
	ErrHTTPBanned    = errors.New("http guard: banned")
	ErrHTTPRateLimit = errors.New("http guard: too many requests")
)

type httpClientState struct {
	timestamps  []time.Time
	bannedUntil time.Time
	lastSeen    time.Time
}

type httpGuardShard struct {
	mu      sync.Mutex
	clients map[string]*httpClientState
	ops     uint64
}

type HTTPGuard struct {
	shards [httpLimiterShardCount]httpGuardShard

	eventMu     sync.Mutex
	nextEventAt time.Time
}

var defaultHTTPGuard = NewHTTPGuard()

func NewHTTPGuard() *HTTPGuard {
	guard := &HTTPGuard{}

	for index := range guard.shards {
		guard.shards[index].clients = make(map[string]*httpClientState)
	}

	return guard
}

func httpLimiterShardIndex(key string) int {
	hash := uint64(14695981039346656037)

	for index := 0; index < len(key); index++ {
		hash ^= uint64(key[index])
		hash *= uint64(1099511628211)
	}

	return int(hash % uint64(httpLimiterShardCount))
}

func pruneHTTPGuardShardLocked(
	shard *httpGuardShard,
	now time.Time,
) {
	if shard == nil || len(shard.clients) == 0 {
		return
	}

	cutoff := now.Add(-httpGuardStateRetention)

	for key, state := range shard.clients {
		if state == nil {
			delete(shard.clients, key)
			continue
		}

		if !state.bannedUntil.IsZero() &&
			now.Before(state.bannedUntil) {
			continue
		}

		if state.lastSeen.IsZero() ||
			state.lastSeen.Before(cutoff) {
			delete(shard.clients, key)
		}
	}
}

func evictOldestHTTPGuardKeyLocked(
	shard *httpGuardShard,
	now time.Time,
) {
	if shard == nil || len(shard.clients) == 0 {
		return
	}

	oldestKey := ""
	var oldestSeen time.Time

	for key, state := range shard.clients {
		if state == nil {
			delete(shard.clients, key)
			return
		}

		if !state.bannedUntil.IsZero() &&
			now.Before(state.bannedUntil) {
			continue
		}

		if oldestKey == "" ||
			state.lastSeen.Before(oldestSeen) {
			oldestKey = key
			oldestSeen = state.lastSeen
		}
	}

	if oldestKey != "" {
		delete(shard.clients, oldestKey)
		return
	}

	// If every retained record is currently banned, bounded memory still
	// wins over indefinite key growth.
	for key, state := range shard.clients {
		if state == nil {
			delete(shard.clients, key)
			return
		}

		if oldestKey == "" ||
			state.lastSeen.Before(oldestSeen) {
			oldestKey = key
			oldestSeen = state.lastSeen
		}
	}

	if oldestKey != "" {
		delete(shard.clients, oldestKey)
	}
}

func (g *HTTPGuard) shouldLogEvent(
	now time.Time,
) bool {
	if g == nil {
		return false
	}

	g.eventMu.Lock()
	defer g.eventMu.Unlock()

	if !g.nextEventAt.IsZero() &&
		now.Before(g.nextEventAt) {
		return false
	}

	g.nextEventAt = now.Add(httpSecurityEventCooldown)
	return true
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
	if g == nil || ip == "" {
		return nil
	}

	now := time.Now()
	windowStart := now.Add(-httpRateWindow)

	shard := &g.shards[httpLimiterShardIndex(ip)]

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if shard.clients == nil {
		shard.clients = make(map[string]*httpClientState)
	}

	shard.ops++

	if shard.ops%httpLimiterSweepEvery == 0 {
		pruneHTTPGuardShardLocked(
			shard,
			now,
		)
	}

	state, ok := shard.clients[ip]

	if !ok {
		if len(shard.clients) >= httpGuardMaxKeysPerShard {
			pruneHTTPGuardShardLocked(
				shard,
				now,
			)
		}

		if len(shard.clients) >= httpGuardMaxKeysPerShard {
			evictOldestHTTPGuardKeyLocked(
				shard,
				now,
			)
		}

		state = &httpClientState{
			lastSeen: now,
		}

		shard.clients[ip] = state
	} else {
		state.lastSeen = now
	}

	if !state.bannedUntil.IsZero() &&
		now.Before(state.bannedUntil) {
		return ErrHTTPBanned
	}

	if !state.bannedUntil.IsZero() &&
		!now.Before(state.bannedUntil) {
		state.bannedUntil = time.Time{}
	}

	h := state.timestamps
	j := 0

	for _, timestamp := range h {
		if timestamp.After(windowStart) {
			h[j] = timestamp
			j++
		}
	}

	h = h[:j]
	h = append(h, now)

	state.timestamps = h
	state.lastSeen = now

	if len(h) > httpMaxRequestsPerIPInWindow {
		state.bannedUntil = now.Add(httpBanDuration)
		return ErrHTTPRateLimit
	}

	return nil
}

func writeRequestBodyTooLargeV1(
	w http.ResponseWriter,
	err error,
) bool {
	if err == nil {
		return false
	}

	var maxBytesErr *http.MaxBytesError

	if !errors.As(
		err,
		&maxBytesErr,
	) {
		return false
	}

	writeError(
		w,
		http.StatusRequestEntityTooLarge,
		"request body too large",
	)

	return true
}

// Wrap documents the QuantumCoin component used by this module.
// Tüm HTTP API'yi sarmak için middleware
func (g *HTTPGuard) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIPFromRequest(r)

		if err := g.allow(ip); err != nil {
			logEvent := g.shouldLogEvent(time.Now())

			switch err {
			case ErrHTTPBanned:
				if logEvent {
					_ = security.AppendEvent(".", security.NewEvent(
						"HTTP_BANNED_REQUEST",
						security.SeverityHigh,
						"http_guard",
						ip,
						"Blocked request from temporarily banned IP.",
						map[string]any{
							"path":   r.URL.Path,
							"method": r.Method,
						},
					))
				}

				writeError(w, http.StatusForbidden, "Forbidden")

			case ErrHTTPRateLimit:
				if logEvent {
					_ = security.AppendEvent(".", security.NewEvent(
						"HTTP_RATE_LIMIT_TRIGGERED",
						security.SeverityWarning,
						"http_guard",
						ip,
						"HTTP rate limit triggered and temporary ban applied.",
						map[string]any{
							"path":   r.URL.Path,
							"method": r.Method,
							"window": httpRateWindow.String(),
							"limit":  httpMaxRequestsPerIPInWindow,
						},
					))
				}

				writeError(w, http.StatusTooManyRequests, "Too Many Requests")

			default:
				if logEvent {
					_ = security.AppendEvent(".", security.NewEvent(
						"HTTP_GUARD_ERROR",
						security.SeverityWarning,
						"http_guard",
						ip,
						err.Error(),
						map[string]any{
							"path":   r.URL.Path,
							"method": r.Method,
						},
					))
				}

				writeError(w, http.StatusServiceUnavailable, "Service Unavailable")
			}

			return
		}

		if r.ContentLength > httpMaxBodyBytes {
			writeError(
				w,
				http.StatusRequestEntityTooLarge,
				"request body too large",
			)
			return
		}

		r.Body = http.MaxBytesReader(
			w,
			r.Body,
			httpMaxBodyBytes,
		)

		next.ServeHTTP(w, r)
	})
}

//
// === KRİTİK ENDPOINT LİMİTLEYİCİ (uç bazlı ince ayar) ===
//

type endpointHitState struct {
	timestamps []time.Time
	lastSeen   time.Time
}

type endpointLimiterShard struct {
	mu   sync.Mutex
	hits map[string]*endpointHitState
	ops  uint64
}

type EndpointLimiter struct {
	shards [httpLimiterShardCount]endpointLimiterShard

	eventMu     sync.Mutex
	nextEventAt time.Time
}

func NewEndpointLimiter() *EndpointLimiter {
	limiter := &EndpointLimiter{}

	for index := range limiter.shards {
		limiter.shards[index].hits = make(map[string]*endpointHitState)
	}

	return limiter
}

func endpointLimiterRetention(
	window time.Duration,
) time.Duration {
	retention := 2 * window

	if retention < endpointLimiterMinRetention {
		retention = endpointLimiterMinRetention
	}

	return retention
}

func pruneEndpointLimiterShardLocked(
	shard *endpointLimiterShard,
	now time.Time,
	window time.Duration,
) {
	if shard == nil || len(shard.hits) == 0 {
		return
	}

	cutoff := now.Add(-endpointLimiterRetention(window))

	for key, state := range shard.hits {
		if state == nil ||
			state.lastSeen.IsZero() ||
			state.lastSeen.Before(cutoff) {
			delete(shard.hits, key)
		}
	}
}

func evictOldestEndpointLimiterKeyLocked(
	shard *endpointLimiterShard,
) {
	if shard == nil || len(shard.hits) == 0 {
		return
	}

	oldestKey := ""
	var oldestSeen time.Time

	for key, state := range shard.hits {
		if state == nil {
			delete(shard.hits, key)
			return
		}

		if oldestKey == "" ||
			state.lastSeen.Before(oldestSeen) {
			oldestKey = key
			oldestSeen = state.lastSeen
		}
	}

	if oldestKey != "" {
		delete(shard.hits, oldestKey)
	}
}

func (el *EndpointLimiter) shouldLogEvent(
	now time.Time,
) bool {
	if el == nil {
		return false
	}

	el.eventMu.Lock()
	defer el.eventMu.Unlock()

	if !el.nextEventAt.IsZero() &&
		now.Before(el.nextEventAt) {
		return false
	}

	el.nextEventAt = now.Add(httpSecurityEventCooldown)
	return true
}

// Global kritik limiter
var criticalLimiter = NewEndpointLimiter()

// ip + endpoint key bazında pencere/limit kontrolü
func (el *EndpointLimiter) allow(
	ip string,
	key string,
	window time.Duration,
	max int,
) bool {
	if el == nil ||
		ip == "" ||
		key == "" ||
		max <= 0 {
		return true
	}

	now := time.Now()
	windowStart := now.Add(-window)
	compoundKey := ip + "|" + key

	shard := &el.shards[httpLimiterShardIndex(compoundKey)]

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if shard.hits == nil {
		shard.hits = make(map[string]*endpointHitState)
	}

	shard.ops++

	if shard.ops%httpLimiterSweepEvery == 0 {
		pruneEndpointLimiterShardLocked(
			shard,
			now,
			window,
		)
	}

	state, ok := shard.hits[compoundKey]

	if !ok {
		if len(shard.hits) >= endpointLimiterMaxKeysPerShard {
			pruneEndpointLimiterShardLocked(
				shard,
				now,
				window,
			)
		}

		if len(shard.hits) >= endpointLimiterMaxKeysPerShard {
			evictOldestEndpointLimiterKeyLocked(
				shard,
			)
		}

		state = &endpointHitState{
			lastSeen: now,
		}

		shard.hits[compoundKey] = state
	}

	h := state.timestamps
	j := 0

	for _, timestamp := range h {
		if timestamp.After(windowStart) {
			h[j] = timestamp
			j++
		}
	}

	h = h[:j]
	h = append(h, now)

	state.timestamps = h
	state.lastSeen = now

	return len(h) <= max
}

// WrapFunc documents the QuantumCoin component used by this module.
// Belirli bir endpoint için HandlerFunc saran yardımcı
func (el *EndpointLimiter) WrapFunc(
	key string,
	window time.Duration,
	max int,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIPFromRequest(r)

		if !el.allow(
			ip,
			key,
			window,
			max,
		) {
			if el.shouldLogEvent(time.Now()) {
				_ = security.AppendEvent(".", security.NewEvent(
					"ENDPOINT_RATE_LIMIT_TRIGGERED",
					security.SeverityWarning,
					"endpoint_limiter",
					ip,
					"Critical endpoint rate limit triggered.",
					map[string]any{
						"endpointKey": key,
						"path":        r.URL.Path,
						"method":      r.Method,
						"window":      window.String(),
						"limit":       max,
					},
				))
			}

			writeError(w, http.StatusTooManyRequests, "Too Many Requests")

			return
		}

		next(w, r)
	}
}
