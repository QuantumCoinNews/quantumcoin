package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func clearCORSTestEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		"QC_API_TOKEN",
		"QC_PUBLIC_TESTNET",
		"QC_API_PUBLIC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_TESTNET",
		"QC_NET_MAGIC",
		"QC_NETWORK",
		"QC_NETWORK_MODE",
		"QC_CHAIN_MODE",
		"QC_MODE",
		"QC_API_LISTEN",
		"QC_CORS_ALLOWED_ORIGINS",
		"QC_CORS_ALLOWED_ORIGIN",
	}

	old := map[string]string{}
	had := map[string]bool{}

	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			old[k] = v
			had[k] = true
		}
		_ = os.Unsetenv(k)
	}

	t.Cleanup(func() {
		for _, k := range keys {
			if had[k] {
				_ = os.Setenv(k, old[k])
			} else {
				_ = os.Unsetenv(k)
			}
		}
	})
}

func TestWithCORSDevModeKeepsWildcard(t *testing.T) {
	clearCORSTestEnv(t)

	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected dev wildcard CORS, got %q", got)
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestWithCORSPublicTestnetRejectsUnknownPreflight(t *testing.T) {
	clearCORSTestEnv(t)
	t.Setenv("QC_PUBLIC_TESTNET", "1")

	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS allow-origin for unknown public origin, got %q", got)
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unknown public preflight, got %d", rr.Code)
	}
}

func TestWithCORSPublicTestnetAllowsConfiguredOrigin(t *testing.T) {
	clearCORSTestEnv(t)
	t.Setenv("QC_PUBLIC_TESTNET", "1")
	t.Setenv("QC_CORS_ALLOWED_ORIGINS", "https://wallet.quantumcoin.test,https://game.quantumcoin.test")

	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "https://wallet.quantumcoin.test")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://wallet.quantumcoin.test" {
		t.Fatalf("expected configured origin, got %q", got)
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for allowed public preflight, got %d", rr.Code)
	}
}
func TestWithCORSPublicTestnetRejectsUnknownNonPreflight(
	t *testing.T,
) {
	clearCORSTestEnv(t)
	t.Setenv("QC_PUBLIC_TESTNET", "1")

	for _, method := range []string{
		http.MethodGet,
		http.MethodPost,
	} {
		t.Run(method, func(t *testing.T) {
			handlerCalled := false

			h := withCORS(http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					handlerCalled = true
					w.WriteHeader(http.StatusOK)
				},
			))

			req := httptest.NewRequest(
				method,
				"/api/side-effect",
				nil,
			)

			req.Header.Set(
				"Origin",
				"https://evil.example",
			)

			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if handlerCalled {
				t.Fatalf(
					"disallowed public origin reached handler for %s",
					method,
				)
			}

			if rr.Code != http.StatusForbidden {
				t.Fatalf(
					"status=%d want=%d",
					rr.Code,
					http.StatusForbidden,
				)
			}

			if got := rr.Header().Get(
				"Access-Control-Allow-Origin",
			); got != "" {
				t.Fatalf(
					"unexpected allow-origin %q",
					got,
				)
			}
		})
	}
}

func TestWithCORSPublicTestnetAllowsConfiguredNonPreflight(
	t *testing.T,
) {
	clearCORSTestEnv(t)

	t.Setenv(
		"QC_PUBLIC_TESTNET",
		"1",
	)

	t.Setenv(
		"QC_CORS_ALLOWED_ORIGINS",
		"https://wallet.quantumcoin.test",
	)

	for _, method := range []string{
		http.MethodGet,
		http.MethodPost,
	} {
		t.Run(method, func(t *testing.T) {
			handlerCalled := false

			h := withCORS(http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					handlerCalled = true
					w.WriteHeader(http.StatusAccepted)
				},
			))

			req := httptest.NewRequest(
				method,
				"/api/allowed",
				nil,
			)

			req.Header.Set(
				"Origin",
				"https://wallet.quantumcoin.test",
			)

			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if !handlerCalled {
				t.Fatalf(
					"configured origin did not reach handler for %s",
					method,
				)
			}

			if rr.Code != http.StatusAccepted {
				t.Fatalf(
					"status=%d want=%d",
					rr.Code,
					http.StatusAccepted,
				)
			}

			if got := rr.Header().Get(
				"Access-Control-Allow-Origin",
			); got != "https://wallet.quantumcoin.test" {
				t.Fatalf(
					"allow-origin=%q",
					got,
				)
			}
		})
	}
}

func TestWithCORSPublicTestnetAllowsRequestWithoutOrigin(
	t *testing.T,
) {
	clearCORSTestEnv(t)
	t.Setenv("QC_PUBLIC_TESTNET", "1")

	handlerCalled := false

	h := withCORS(http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		},
	))

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/server-to-server",
		nil,
	)

	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Fatal(
			"request without Origin must remain usable by non-browser clients",
		)
	}

	if rr.Code != http.StatusOK {
		t.Fatalf(
			"status=%d want=%d",
			rr.Code,
			http.StatusOK,
		)
	}

	if got := rr.Header().Get(
		"Access-Control-Allow-Origin",
	); got != "" {
		t.Fatalf(
			"request without Origin unexpectedly received allow-origin %q",
			got,
		)
	}
}
