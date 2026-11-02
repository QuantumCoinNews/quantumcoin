package main

import (
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestHTTPGuardStateBoundedUnderIPChurn(
	t *testing.T,
) {
	guard := NewHTTPGuard()

	total :=
		httpLimiterShardCount *
			(httpGuardMaxKeysPerShard + 64)

	for index := 0; index < total; index++ {
		err := guard.allow(
			"guard-churn-" + strconv.Itoa(index),
		)

		if err != nil {
			t.Fatalf(
				"unexpected allow error at %d: %v",
				index,
				err,
			)
		}
	}

	totalKeys := 0

	for index := range guard.shards {
		shard := &guard.shards[index]

		shard.mu.Lock()
		size := len(shard.clients)
		shard.mu.Unlock()

		totalKeys += size

		if size > httpGuardMaxKeysPerShard {
			t.Fatalf(
				"shard=%d clients=%d cap=%d",
				index,
				size,
				httpGuardMaxKeysPerShard,
			)
		}
	}

	maxTotal :=
		httpLimiterShardCount *
			httpGuardMaxKeysPerShard

	if totalKeys > maxTotal {
		t.Fatalf(
			"total clients=%d cap=%d",
			totalKeys,
			maxTotal,
		)
	}
}

func TestEndpointLimiterStateBoundedUnderKeyChurn(
	t *testing.T,
) {
	limiter := NewEndpointLimiter()

	total :=
		httpLimiterShardCount *
			(endpointLimiterMaxKeysPerShard + 64)

	for index := 0; index < total; index++ {
		allowed := limiter.allow(
			"endpoint-churn-"+strconv.Itoa(index),
			"tx_send",
			10*time.Second,
			5,
		)

		if !allowed {
			t.Fatalf(
				"unexpected rejection at %d",
				index,
			)
		}
	}

	totalKeys := 0

	for index := range limiter.shards {
		shard := &limiter.shards[index]

		shard.mu.Lock()
		size := len(shard.hits)
		shard.mu.Unlock()

		totalKeys += size

		if size > endpointLimiterMaxKeysPerShard {
			t.Fatalf(
				"shard=%d hits=%d cap=%d",
				index,
				size,
				endpointLimiterMaxKeysPerShard,
			)
		}
	}

	maxTotal :=
		httpLimiterShardCount *
			endpointLimiterMaxKeysPerShard

	if totalKeys > maxTotal {
		t.Fatalf(
			"total endpoint keys=%d cap=%d",
			totalKeys,
			maxTotal,
		)
	}
}

func TestHTTPGuardPrunesExpiredStateAndKeepsActiveBan(
	t *testing.T,
) {
	guard := NewHTTPGuard()
	now := time.Now()

	shard := &guard.shards[0]

	shard.mu.Lock()

	shard.clients["stale"] = &httpClientState{
		lastSeen: now.Add(
			-httpGuardStateRetention -
				time.Second,
		),
	}

	shard.clients["active-ban"] = &httpClientState{
		lastSeen: now.Add(
			-httpGuardStateRetention -
				time.Second,
		),
		bannedUntil: now.Add(time.Minute),
	}

	pruneHTTPGuardShardLocked(
		shard,
		now,
	)

	_, staleExists :=
		shard.clients["stale"]

	_, activeBanExists :=
		shard.clients["active-ban"]

	shard.mu.Unlock()

	if staleExists {
		t.Fatal(
			"expired HTTPGuard state was not evicted",
		)
	}

	if !activeBanExists {
		t.Fatal(
			"active temporary ban was evicted",
		)
	}
}

func TestEndpointLimiterPrunesExpiredState(
	t *testing.T,
) {
	limiter := NewEndpointLimiter()
	now := time.Now()

	shard := &limiter.shards[0]

	shard.mu.Lock()

	shard.hits["stale"] = &endpointHitState{
		lastSeen: now.Add(
			-2 * endpointLimiterMinRetention,
		),
	}

	shard.hits["fresh"] = &endpointHitState{
		lastSeen: now,
	}

	pruneEndpointLimiterShardLocked(
		shard,
		now,
		10*time.Second,
	)

	_, staleExists :=
		shard.hits["stale"]

	_, freshExists :=
		shard.hits["fresh"]

	shard.mu.Unlock()

	if staleExists {
		t.Fatal(
			"expired endpoint state was not evicted",
		)
	}

	if !freshExists {
		t.Fatal(
			"fresh endpoint state was evicted",
		)
	}
}

func TestHTTPGuardRateLimitSemanticsPreserved(
	t *testing.T,
) {
	guard := NewHTTPGuard()

	for index := 0; index < httpMaxRequestsPerIPInWindow; index++ {
		err := guard.allow("same-ip")

		if err != nil {
			t.Fatalf(
				"request %d rejected early: %v",
				index,
				err,
			)
		}
	}

	err := guard.allow("same-ip")

	if !errors.Is(
		err,
		ErrHTTPRateLimit,
	) {
		t.Fatalf(
			"threshold error=%v want=%v",
			err,
			ErrHTTPRateLimit,
		)
	}

	err = guard.allow("same-ip")

	if !errors.Is(
		err,
		ErrHTTPBanned,
	) {
		t.Fatalf(
			"post-threshold error=%v want=%v",
			err,
			ErrHTTPBanned,
		)
	}
}

func TestEndpointLimiterRateLimitSemanticsPreserved(
	t *testing.T,
) {
	limiter := NewEndpointLimiter()

	if !limiter.allow(
		"same-ip",
		"tx_send",
		10*time.Second,
		2,
	) {
		t.Fatal("first request rejected")
	}

	if !limiter.allow(
		"same-ip",
		"tx_send",
		10*time.Second,
		2,
	) {
		t.Fatal("second request rejected")
	}

	if limiter.allow(
		"same-ip",
		"tx_send",
		10*time.Second,
		2,
	) {
		t.Fatal(
			"third request should be rate limited",
		)
	}
}

func TestHTTPGuardEventEmissionIsCoalesced(
	t *testing.T,
) {
	guard := NewHTTPGuard()

	base := time.Unix(1000, 0)

	if !guard.shouldLogEvent(base) {
		t.Fatal(
			"first event should be emitted",
		)
	}

	if guard.shouldLogEvent(
		base.Add(
			httpSecurityEventCooldown / 2,
		),
	) {
		t.Fatal(
			"event inside cooldown should be suppressed",
		)
	}

	if !guard.shouldLogEvent(
		base.Add(
			httpSecurityEventCooldown,
		),
	) {
		t.Fatal(
			"event after cooldown should be emitted",
		)
	}
}

func TestEndpointLimiterEventEmissionIsCoalesced(
	t *testing.T,
) {
	limiter := NewEndpointLimiter()

	base := time.Unix(2000, 0)

	if !limiter.shouldLogEvent(base) {
		t.Fatal(
			"first endpoint event should be emitted",
		)
	}

	if limiter.shouldLogEvent(
		base.Add(
			httpSecurityEventCooldown / 2,
		),
	) {
		t.Fatal(
			"endpoint event inside cooldown should be suppressed",
		)
	}

	if !limiter.shouldLogEvent(
		base.Add(
			httpSecurityEventCooldown,
		),
	) {
		t.Fatal(
			"endpoint event after cooldown should be emitted",
		)
	}
}

func TestHTTPRateLimiterConcurrentChurn(
	t *testing.T,
) {
	guard := NewHTTPGuard()
	limiter := NewEndpointLimiter()

	const (
		workers    = 64
		iterations = 200
	)

	var wait sync.WaitGroup

	wait.Add(workers)

	for worker := 0; worker < workers; worker++ {
		worker := worker

		go func() {
			defer wait.Done()

			for index := 0; index < iterations; index++ {
				ip :=
					"concurrent-" +
						strconv.Itoa(worker) +
						"-" +
						strconv.Itoa(index)

				_ = guard.allow(ip)

				_ = limiter.allow(
					ip,
					"endpoint-"+strconv.Itoa(index%4),
					10*time.Second,
					5,
				)
			}
		}()
	}

	wait.Wait()

	for index := range guard.shards {
		shard := &guard.shards[index]

		shard.mu.Lock()
		size := len(shard.clients)
		shard.mu.Unlock()

		if size > httpGuardMaxKeysPerShard {
			t.Fatalf(
				"guard shard=%d size=%d cap=%d",
				index,
				size,
				httpGuardMaxKeysPerShard,
			)
		}
	}

	for index := range limiter.shards {
		shard := &limiter.shards[index]

		shard.mu.Lock()
		size := len(shard.hits)
		shard.mu.Unlock()

		if size > endpointLimiterMaxKeysPerShard {
			t.Fatalf(
				"endpoint shard=%d size=%d cap=%d",
				index,
				size,
				endpointLimiterMaxKeysPerShard,
			)
		}
	}
}
