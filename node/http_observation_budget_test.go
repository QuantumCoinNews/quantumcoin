package node

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"quantumcoin/monitoring"
	"quantumcoin/security"
)

func TestHTTPObservationFloodBoundsDistinctIPNotifications(t *testing.T) {
	previous := security.LiveEvidence
	security.LiveEvidence = &security.RuntimeEvidence{}
	t.Cleanup(func() { security.LiveEvidence = previous })
	runtime := NewRuntime(nil, nil)
	start := time.Now()
	for i := range 40 {
		ip := "198.18.0." + strconv.Itoa(i+1)
		for _, status := range []int{401, 429, 500} {
			runtime.RecordHTTPObservation(ip, "/api/test", "POST", status)
		}
	}
	maximum := 3 * (int(time.Since(start)/time.Second) + 1)
	state := runtime.Monitoring().Snapshot()
	if len(state.Events) > maximum {
		t.Fatalf("HTTP flood amplified monitoring writes: events=%d budget=%d", len(state.Events), maximum)
	}
	categories := map[string]bool{}
	for _, event := range state.Events {
		categories[event.Category] = true
		if event.Category == "api_failure" && event.Severity != monitoring.SeverityCritical {
			t.Fatal("critical category demoted")
		}
	}
	if !categories["auth_failure"] || !categories["action_rate_limit"] || !categories["api_failure"] {
		t.Fatal("one category suppressed another", categories)
	}
	reports := security.LiveEvidence.Reports(false, false)
	if reports["actionBurstGuard"]["evidenceSequence"] != uint64(120) {
		t.Fatal("coalescing discarded live evidence", reports["actionBurstGuard"]["evidenceSequence"])
	}
}

func TestHTTPObservationStalledBackendDoesNotQueue(t *testing.T) {
	runtime := NewRuntime(nil, nil)
	runtime.mu.Lock()
	var unlock sync.Once
	release := func() { unlock.Do(runtime.mu.Unlock) }
	defer release()
	first, second := make(chan struct{}), make(chan struct{})
	go func() { runtime.RecordHTTPObservation("198.18.1.1", "/api/test", "GET", 401); close(first) }()
	defer func() {
		release()
		select {
		case <-first:
		case <-time.After(3 * time.Second):
			t.Error("monitoring writer did not finish")
		}
	}()
	budget := &runtime.httpMonitorBudgets[0]
	deadline := time.Now().Add(3 * time.Second)
	for {
		budget.mu.Lock()
		ready := budget.busy
		if ready {
			budget.nextAt = time.Time{}
		}
		budget.mu.Unlock()
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("monitoring writer not reserved")
		}
		time.Sleep(time.Millisecond)
	}
	go func() { runtime.RecordHTTPObservation("198.18.1.2", "/api/test", "GET", 401); close(second) }()
	defer func() {
		release()
		select {
		case <-second:
		case <-time.After(3 * time.Second):
			t.Error("coalesced observation did not finish")
		}
	}()
	select {
	case <-second:
	case <-time.After(2 * time.Second):
		t.Fatal("observation queued behind stalled backend")
	}
	release()
	select {
	case <-first:
	case <-time.After(3 * time.Second):
		t.Fatal("monitor did not recover")
	}
	runtime.RecordHTTPObservation("198.18.1.3", "/api/test", "GET", 401)
	if len(runtime.Monitoring().Snapshot().Events) != 2 {
		t.Fatal("monitoring budget did not recover")
	}
}
