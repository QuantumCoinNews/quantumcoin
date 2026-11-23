//go:build windows

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/windows"
	"quantumcoin/security"
)

func TestLimiterSlowLogSinkDoesNotQueueRejectedRequests(t *testing.T) {
	if mode := os.Getenv("QC_TEST_LIMITER_LOG_CHILD_V1"); mode != "" {
		limiterSlowLogChildV1(t, mode)
		return
	}
	for _, mode := range []string{"guard", "endpoint"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLimiterSlowLogSinkDoesNotQueueRejectedRequests$", "-test.timeout=20s")
			command.Env = append(os.Environ(), "QC_TEST_LIMITER_LOG_CHILD_V1="+mode, "QC_TEST_LIMITER_LOG_DIR_V1="+t.TempDir())
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("isolated slow-log acceptance failed: %v\n%s", err, output)
			}
		})
	}
}

func limiterSlowLogChildV1(t *testing.T, mode string) {
	if err := security.ConfigureEventDirectory(filepath.Join(os.Getenv("QC_TEST_LIMITER_LOG_DIR_V1"), "chain.dat"), "local-release"); err != nil {
		t.Fatal(err)
	}
	dir := security.EventDirectory()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(filepath.Join(dir, ".security-record.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var overlap windows.Overlapped
	if err := windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlap); err != nil {
		t.Fatal(err)
	}
	var unlock sync.Once
	release := func() { unlock.Do(func() { _ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlap) }) }
	defer release()
	guard, limiter := NewHTTPGuard(), NewEndpointLimiter()
	const ip = "192.0.2.28"
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("rejected request reached handler") })
	var handler http.Handler
	want := http.StatusForbidden
	if mode == "guard" {
		for range httpMaxRequestsPerIPInWindow + 1 {
			_ = guard.allow(ip)
		}
		handler = guard.Wrap(next)
	} else {
		limiter.allow(ip, "slow-log", time.Hour, 1)
		handler = limiter.WrapFunc("slow-log", time.Hour, 1, next)
		want = http.StatusTooManyRequests
	}
	send := func() int {
		request := httptest.NewRequest(http.MethodGet, "/slow-log", nil)
		request.RemoteAddr = ip + ":1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}
	first, second := make(chan int, 1), make(chan int, 1)
	go func() { first <- send() }()
	defer func() {
		release()
		select {
		case <-first:
		case <-time.After(3 * time.Second):
			t.Error("first log writer did not finish")
		}
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu, nextEvent := &guard.eventMu, &guard.nextEventAt
		if mode == "endpoint" {
			mu, nextEvent = &limiter.eventMu, &limiter.nextEventAt
		}
		mu.Lock()
		ready := !nextEvent.IsZero()
		if ready {
			*nextEvent = time.Time{}
		} // Simulate the next eligible log interval.
		mu.Unlock()
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first rejection did not reserve log slot")
		}
		time.Sleep(time.Millisecond)
	}
	go func() { second <- send() }()
	defer func() {
		release()
		select {
		case <-second:
		case <-time.After(3 * time.Second):
			t.Error("second request did not finish")
		}
	}()
	select {
	case status := <-second:
		second <- status // Preserve completion for deferred cleanup.
		if status != want {
			t.Fatal("rejection status changed", status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second rejected request queued behind stalled log I/O")
	}
	release()
	select {
	case status := <-first:
		first <- status
		if status != want {
			t.Fatal("first rejection status changed", status)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("log writer did not recover")
	}
	if status := send(); status != want {
		t.Fatal("post-recovery rejection changed", status)
	}
	log := security.ReadEvents(".", 100)
	if log.Error != "" || len(log.Events) != 2 {
		t.Fatalf("log suppression/recovery failed: count=%d error=%s", len(log.Events), log.Error)
	}
}
