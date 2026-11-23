package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testFreeAPIPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)
	_ = listener.Close()
	return port
}

func TestBackendCoordinatorRejectsExistingUnrelatedOrUnauthorizedService(t *testing.T) {
	for _, mode := range []string{"unrelated", "unauthorized"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if mode == "unauthorized" {
					w.Header().Set("WWW-Authenticate", HealthAuthRealmV1)
					w.WriteHeader(401)
					fmt.Fprint(w, `{"success":false}`)
				} else {
					fmt.Fprint(w, `{"ok":true}`)
				}
			}))
			defer server.Close()
			origin, _ := url.Parse(server.URL)
			started := false
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := EnsureLocalBackendV1(ctx, origin.Port(), "", func() (BackendHandleV1, error) { started = true; return BackendHandleV1{}, nil })
			want := ErrHealthContractV1
			if mode == "unauthorized" {
				want = ErrHealthUnauthorizedV1
			}
			if !errors.Is(err, want) || started {
				t.Fatalf("existing service caused startup: %v", err)
			}
			response, err := http.Get(server.URL)
			if err != nil {
				t.Fatal("existing service was disturbed")
			}
			_ = response.Body.Close()
		})
	}
}

func TestBackendCoordinatorConcurrentClientsStartOneDelayedBackend(t *testing.T) {
	port := testFreeAPIPort(t)
	var starts, stops, releases atomic.Int32
	var server *http.Server
	var ready atomic.Bool
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	start := func() (BackendHandleV1, error) {
		starts.Add(1)
		listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
		if err != nil {
			return BackendHandleV1{}, err
		}
		server = &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != CanonicalHealthPathV1 || r.Header.Get("X-QC-API-KEY") != "fixture-token" {
				http.Error(w, "unexpected request", 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if !ready.Load() {
				w.WriteHeader(503)
				fmt.Fprint(w, `{"success":false}`)
				return
			}
			fmt.Fprintf(w, `{"ok":true,"service":%q,"healthVersion":%q}`, HealthServiceV1, HealthProtocolV1)
		})}
		go func() { _ = server.Serve(listener) }()
		time.AfterFunc(120*time.Millisecond, func() { ready.Store(true) })
		return BackendHandleV1{Stop: func() { stops.Add(1); _ = server.Close() }, Release: func() error { releases.Add(1); return nil }}, nil
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { results <- EnsureLocalBackendV1(ctx, port, "fixture-token", start) })
	}
	wg.Wait()
	close(results)
	if server != nil {
		defer server.Close()
	}
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if starts.Load() != 1 || stops.Load() != 0 || releases.Load() != 1 {
		t.Fatalf("start/stop/release=%d/%d/%d", starts.Load(), stops.Load(), releases.Load())
	}
}

func TestBackendCoordinatorStopsOnlyOwnedUnhealthyBackend(t *testing.T) {
	port := testFreeAPIPort(t)
	var stopped, released atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()
	err := EnsureLocalBackendV1(ctx, port, "fixture-token", func() (BackendHandleV1, error) {
		listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
		if err != nil {
			return BackendHandleV1{}, err
		}
		server := &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
		})}
		go func() { _ = server.Serve(listener) }()
		return BackendHandleV1{Stop: func() { stopped.Add(1); _ = server.Close() }, Release: func() error { released.Add(1); return nil }}, nil
	})
	if !errors.Is(err, context.DeadlineExceeded) || stopped.Load() != 1 || released.Load() != 0 {
		t.Fatalf("timeout cleanup: %v stop/release=%d/%d", err, stopped.Load(), released.Load())
	}
}

func TestStartupPortNormalizationAndInputValidation(t *testing.T) {
	for _, input := range []string{"8082", ":8082", " 08082 "} {
		if got, err := NormalizeAPIPortV1(input); err != nil || got != "8082" {
			t.Fatal(input, got, err)
		}
	}
	for _, input := range []string{"", "0", "65536", "8082/path", "8082@remote", "-1"} {
		if _, err := NormalizeAPIPortV1(input); err == nil {
			t.Fatal("invalid port accepted", input)
		}
	}
}

func TestConfiguredAPIPortRejectsLegacyRedirect(t *testing.T) {
	t.Setenv("QC_HTTP_PORT", "")
	t.Setenv("HTTP_PORT", "")
	if port, err := ConfiguredAPIPortV1(); err != nil || port != "8082" {
		t.Fatal(port, err)
	}
	t.Setenv("HTTP_PORT", "8081")
	if _, err := ConfiguredAPIPortV1(); err == nil {
		t.Fatal("legacy override accepted")
	}
	t.Setenv("QC_HTTP_PORT", ":08081")
	if port, err := ConfiguredAPIPortV1(); err != nil || port != "8081" {
		t.Fatal(port, err)
	}
	t.Setenv("QC_HTTP_PORT", "bad")
	if _, err := ConfiguredAPIPortV1(); err == nil {
		t.Fatal("bad canonical port accepted")
	}
}
