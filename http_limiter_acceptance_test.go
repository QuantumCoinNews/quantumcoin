package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func limiterCollisionKeysV1(endpoint string, count int) []string {
	keys := make([]string, 0, count)
	for i := 0; len(keys) < count; i++ {
		ip := "198.18." + strconv.Itoa((i>>8)&255) + "." + strconv.Itoa(i&255)
		compound := ip
		if endpoint != "" {
			compound += "|" + endpoint
		}
		if httpLimiterShardIndex(compound) == 0 {
			keys = append(keys, ip)
		}
	}
	return keys
}

func TestLimiterHTTPCapacityRejectsBeforeHandlerAndKeepsKnownClients(t *testing.T) {
	for _, mode := range []string{"guard", "endpoint"} {
		t.Run(mode, func(t *testing.T) {
			guard, endpoint := NewHTTPGuard(), NewEndpointLimiter()
			guard.nextEventAt = time.Now().Add(24 * time.Hour)
			endpoint.nextEventAt = time.Now().Add(24 * time.Hour)
			key := ""
			if mode == "endpoint" {
				key = "send"
			}
			ips := limiterCollisionKeysV1(key, 257)
			for _, ip := range ips[:256] {
				if mode == "guard" {
					if err := guard.allow(ip); err != nil {
						t.Fatal(err)
					}
				} else if !endpoint.allow(ip, key, time.Hour, 2) {
					t.Fatal("initial quota denied")
				}
			}
			calls := 0
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; w.WriteHeader(http.StatusNoContent) })
			var handler http.Handler = guard.Wrap(next)
			if mode == "endpoint" {
				handler = endpoint.WrapFunc(key, time.Hour, 2, next)
			}
			forwarded := "198.19.0.1"
			for httpLimiterShardIndex(forwarded+map[bool]string{true: "|send", false: ""}[mode == "endpoint"]) == 0 {
				forwarded += "1"
			}
			send := func(ip string) int {
				r := httptest.NewRequest(http.MethodPost, "/arbitrary-path", nil)
				r.RemoteAddr = ip + ":4321"
				r.Header.Set("X-Forwarded-For", forwarded)
				r.Header.Set("X-Real-IP", forwarded)
				r.Header.Set("Forwarded", "for="+forwarded)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				return w.Code
			}
			if status := send(ips[256]); status != http.StatusTooManyRequests || calls != 0 {
				t.Fatal("capacity/forwarded header bypass", status, calls)
			}
			if status := send(ips[0]); status != http.StatusNoContent || calls != 1 {
				t.Fatal("known client lost remaining quota", status, calls)
			}
		})
	}
}

func BenchmarkLimiterRejectedTraffic(b *testing.B) {
	for _, max := range []int{3, 256} {
		b.Run(strconv.Itoa(max), func(b *testing.B) {
			limiter := NewEndpointLimiter()
			now := time.Unix(1700000000, 0)
			for range max {
				limiter.allowAtV1("peer", "send", time.Hour, max, now)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if limiter.allowAtV1("peer", "send", time.Hour, max, now) {
					b.Fatal("quota bypass")
				}
			}
		})
	}
}

func TestLimiterRejectedFloodDoesNotGrowHistory(t *testing.T) {
	limiter := NewEndpointLimiter()
	for range 3 {
		if !limiter.allow("peer", "send", time.Hour, 3) {
			t.Fatal("initial quota denied")
		}
	}
	for range 25000 {
		if limiter.allow("peer", "send", time.Hour, 3) {
			t.Fatal("rejected flood bypassed quota")
		}
	}
	shard := &limiter.shards[httpLimiterShardIndex("peer|send")]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	state := shard.hits["peer|send"]
	if state == nil {
		t.Fatal("quota record missing")
	}
	if len(state.timestamps) > 3 || cap(state.timestamps) > 6 {
		t.Fatalf("rejected traffic retained unbounded history: len=%d cap=%d", len(state.timestamps), cap(state.timestamps))
	}
}

func TestLimiterExpiredWindowsAndCapacityRecover(t *testing.T) {
	base := time.Unix(1700000000, 0)
	guard := NewHTTPGuard()
	keys := limiterCollisionKeysV1("", httpGuardMaxKeysPerShard+1)
	for _, key := range keys[:len(keys)-1] {
		if err := guard.allowAtV1(key, base); err != nil {
			t.Fatal(err)
		}
	}
	if err := guard.allowAtV1(keys[len(keys)-1], base); !errors.Is(err, ErrHTTPLimiterCapacity) {
		t.Fatal("capacity not enforced", err)
	}
	if err := guard.allowAtV1(keys[len(keys)-1], base.Add(httpRateWindow)); err != nil {
		t.Fatal("expired capacity not reclaimed", err)
	}
	ban := NewHTTPGuard()
	for range httpMaxRequestsPerIPInWindow + 1 {
		_ = ban.allowAtV1("peer", base)
	}
	if !errors.Is(ban.allowAtV1("peer", base.Add(httpBanDuration-time.Nanosecond)), ErrHTTPBanned) {
		t.Fatal("ban ended early")
	}
	if err := ban.allowAtV1("peer", base.Add(httpBanDuration)); err != nil {
		t.Fatal("expired ban not released", err)
	}
	limiter := NewEndpointLimiter()
	for range 3 {
		if !limiter.allowAtV1("peer", "send", time.Hour, 3, base) {
			t.Fatal("quota denied")
		}
	}
	if limiter.allowAtV1("peer", "send", time.Hour, 3, base.Add(time.Hour-time.Nanosecond)) {
		t.Fatal("quota reset early")
	}
	if !limiter.allowAtV1("peer", "send", time.Hour, 3, base.Add(time.Hour)) {
		t.Fatal("expired request window did not recover")
	}
	full := NewEndpointLimiter()
	ips := limiterCollisionKeysV1("send", endpointLimiterMaxKeysPerShard+1)
	for _, ip := range ips[:len(ips)-1] {
		if !full.allowAtV1(ip, "send", time.Hour, 1, base) {
			t.Fatal("quota denied")
		}
	}
	if full.allowAtV1(ips[len(ips)-1], "send", time.Hour, 1, base) {
		t.Fatal("capacity not enforced")
	}
	if !full.allowAtV1(ips[len(ips)-1], "send", time.Hour, 1, base.Add(time.Hour)) {
		t.Fatal("expired endpoint capacity not reclaimed")
	}
}

func TestLimiterCapacityCannotEraseActiveBans(t *testing.T) {
	guard := NewHTTPGuard()
	keys := limiterCollisionKeysV1("", httpGuardMaxKeysPerShard+1)
	for _, ip := range keys[:len(keys)-1] {
		for range httpMaxRequestsPerIPInWindow {
			if err := guard.allow(ip); err != nil {
				t.Fatal(err)
			}
		}
		if err := guard.allow(ip); !errors.Is(err, ErrHTTPRateLimit) {
			t.Fatal("ban not established", err)
		}
	}
	if err := guard.allow(keys[len(keys)-1]); err == nil {
		t.Error("full live shard admitted a new key")
	}
	lost := 0
	for _, ip := range keys[:len(keys)-1] {
		if !errors.Is(guard.allow(ip), ErrHTTPBanned) {
			lost++
		}
	}
	if lost != 0 {
		t.Fatalf("key churn erased %d active bans", lost)
	}
}

func TestLimiterCapacityCannotResetEndpointQuota(t *testing.T) {
	limiter := NewEndpointLimiter()
	keys := limiterCollisionKeysV1("send", endpointLimiterMaxKeysPerShard+1)
	for _, ip := range keys[:len(keys)-1] {
		if !limiter.allow(ip, "send", time.Hour, 1) {
			t.Fatal("initial quota denied")
		}
	}
	if limiter.allow(keys[len(keys)-1], "send", time.Hour, 1) {
		t.Error("full live shard admitted a new key")
	}
	reset := 0
	for _, ip := range keys[:len(keys)-1] {
		if limiter.allow(ip, "send", time.Hour, 1) {
			reset++
		}
	}
	if reset != 0 {
		t.Fatalf("key churn reset %d active quotas", reset)
	}
}

func TestLimiterMixedWindowSweepPreservesLongQuota(t *testing.T) {
	limiter := NewEndpointLimiter()
	if !limiter.allow("peer", "long-window", 2*time.Hour, 1) {
		t.Fatal("initial quota denied")
	}
	shard := &limiter.shards[httpLimiterShardIndex("peer|long-window")]
	shard.mu.Lock()
	pruneEndpointLimiterShardLocked(shard, time.Now().Add(90*time.Second), 10*time.Second)
	shard.mu.Unlock()
	if limiter.allow("peer", "long-window", 2*time.Hour, 1) {
		t.Fatal("short-window cleanup erased a live long-window quota")
	}
}

func TestLimiterPolicyAndKeyBudgetsFailClosed(t *testing.T) {
	for _, test := range []struct {
		name, ip, key string
		window        time.Duration
		max           int
	}{
		{"empty-key", "peer", "", time.Second, 1},
		{"zero-window", "peer", "send", 0, 1},
		{"negative-window", "peer", "send", -time.Second, 1},
		{"overflow-window", "peer", "send", time.Duration(1<<63 - 1), 1},
		{"zero-limit", "peer", "send", time.Second, 0},
		{"huge-limit", "peer", "send", time.Second, 257},
		{"long-ip", strings.Repeat("x", 257), "send", time.Second, 1},
		{"long-key", "peer", strings.Repeat("x", 257), time.Second, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			limiter := NewEndpointLimiter()
			if limiter.allow(test.ip, test.key, test.window, test.max) {
				t.Fatal("invalid or unbounded limiter policy accepted")
			}
			for i := range limiter.shards {
				if len(limiter.shards[i].hits) != 0 {
					t.Fatal("invalid policy retained a key")
				}
			}
		})
	}
	guard := NewHTTPGuard()
	if guard.allow(strings.Repeat("x", 257)) == nil {
		t.Fatal("oversized guard identity accepted")
	}
	for i := range guard.shards {
		if len(guard.shards[i].clients) != 0 {
			t.Fatal("oversized identity retained")
		}
	}
}
