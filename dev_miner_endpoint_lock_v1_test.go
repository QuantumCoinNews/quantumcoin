package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func clearDevMinerEndpointLockEnvV1(t *testing.T) {
	t.Helper()

	keys := []string{
		"QC_DEV_MODE",
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
	}

	old := map[string]string{}
	had := map[string]bool{}

	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			old[key] = value
			had[key] = true
		}

		_ = os.Unsetenv(key)
	}

	t.Cleanup(func() {
		for _, key := range keys {
			if had[key] {
				_ = os.Setenv(
					key,
					old[key],
				)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	})
}

func newMinerControlRequest(
	method string,
	path string,
) *http.Request {
	req := httptest.NewRequest(
		method,
		"http://127.0.0.1:8082"+path,
		nil,
	)

	req.RemoteAddr = "127.0.0.1:12345"
	req.Host = "127.0.0.1:8082"

	return req
}

func configureMinerControlSecurity(
	t *testing.T,
) {
	t.Helper()

	clearDevMinerEndpointLockEnvV1(t)

	t.Setenv(
		"QC_DEV_MODE",
		"1",
	)

	t.Setenv(
		"QC_API_TOKEN",
		"test-miner-control-token",
	)
}

func addMinerControlHeaderToken(
	req *http.Request,
) {
	req.Header.Set(
		"X-QC-API-KEY",
		"test-miner-control-token",
	)
}

func TestDevMinerEndpointLockAllowsLocalDevEnvironment(
	t *testing.T,
) {
	clearDevMinerEndpointLockEnvV1(t)
	t.Setenv("QC_DEV_MODE", "1")

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	if !devMinerEndpointAllowedV1(req) {
		t.Fatal(
			"expected direct loopback dev environment to pass environmental lock",
		)
	}
}

func TestDevMinerEndpointLockRejectsWithoutDevMode(
	t *testing.T,
) {
	clearDevMinerEndpointLockEnvV1(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	if devMinerEndpointAllowedV1(req) {
		t.Fatal(
			"request without QC_DEV_MODE was accepted",
		)
	}
}

func TestDevMinerEndpointLockRejectsRemoteAddress(
	t *testing.T,
) {
	clearDevMinerEndpointLockEnvV1(t)
	t.Setenv("QC_DEV_MODE", "1")

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	req.RemoteAddr = "203.0.113.10:12345"

	if devMinerEndpointAllowedV1(req) {
		t.Fatal(
			"remote request passed loopback environment lock",
		)
	}
}

func TestDevMinerEndpointLockRejectsPublicTestnetEvenLocalDev(
	t *testing.T,
) {
	clearDevMinerEndpointLockEnvV1(t)

	t.Setenv("QC_DEV_MODE", "1")
	t.Setenv("QC_PUBLIC_TESTNET", "1")

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	if devMinerEndpointAllowedV1(req) {
		t.Fatal(
			"public testnet miner control was accepted",
		)
	}
}

func TestMinerControlSecurityAllowsHeaderToken(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	addMinerControlHeaderToken(req)

	rec := httptest.NewRecorder()

	if !requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/start",
	) {
		t.Fatalf(
			"valid direct local miner control rejected: status=%d body=%q",
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestMinerControlSecurityAllowsBearerToken(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/stop",
	)

	req.Header.Set(
		"Authorization",
		"Bearer test-miner-control-token",
	)

	rec := httptest.NewRecorder()

	if !requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/stop",
	) {
		t.Fatalf(
			"valid bearer token rejected: status=%d body=%q",
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestMinerControlSecurityRequiresConfiguredToken(
	t *testing.T,
) {
	clearDevMinerEndpointLockEnvV1(t)
	t.Setenv("QC_DEV_MODE", "1")

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	rec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/start",
	) {
		t.Fatal(
			"miner control accepted an empty QC_API_TOKEN configuration",
		)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}
}

func TestMinerControlSecurityRequiresHeaderToken(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	rec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/start",
	) {
		t.Fatal(
			"miner control accepted request without token header",
		)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}
}

func TestMinerControlSecurityRejectsQueryToken(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start?token=test-miner-control-token",
	)

	rec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/start",
	) {
		t.Fatal(
			"miner control accepted query-string token",
		)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}
}

func TestMinerControlSecurityRequiresPost(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodGet,
		"/api/miner/start",
	)

	addMinerControlHeaderToken(req)

	rec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/start",
	) {
		t.Fatal(
			"GET miner control request was accepted",
		)
	}

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusMethodNotAllowed,
		)
	}

	if got := rec.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf(
			"Allow=%q, want POST",
			got,
		)
	}
}

func TestMinerControlSecurityRejectsCrossOriginBrowser(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	addMinerControlHeaderToken(req)

	req.Header.Set(
		"Origin",
		"https://evil.example",
	)

	req.Header.Set(
		"Sec-Fetch-Site",
		"cross-site",
	)

	rec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/start",
	) {
		t.Fatal(
			"cross-origin browser request was accepted",
		)
	}

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusForbidden,
		)
	}
}

func TestMinerControlSecurityAllowsSameOriginBrowserWithToken(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	addMinerControlHeaderToken(req)

	req.Header.Set(
		"Origin",
		"http://127.0.0.1:8082",
	)

	req.Header.Set(
		"Sec-Fetch-Site",
		"same-origin",
	)

	rec := httptest.NewRecorder()

	if !requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/start",
	) {
		t.Fatalf(
			"same-origin browser request with valid token rejected: status=%d body=%q",
			rec.Code,
			rec.Body.String(),
		)
	}
}

func TestMinerControlSecuritySameOriginBrowserStillRequiresToken(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	req.Header.Set(
		"Origin",
		"http://127.0.0.1:8082",
	)

	req.Header.Set(
		"Sec-Fetch-Site",
		"same-origin",
	)

	rec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/start",
	) {
		t.Fatal(
			"same-origin browser request bypassed required API token",
		)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusUnauthorized,
		)
	}
}

func TestMinerControlSecurityRejectsForwardedProxy(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/stop",
	)

	addMinerControlHeaderToken(req)

	req.Header.Set(
		"X-Forwarded-For",
		"203.0.113.10",
	)

	rec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/stop",
	) {
		t.Fatal(
			"proxied miner control request was accepted",
		)
	}

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusForbidden,
		)
	}
}

func TestMinerControlSecurityRejectsForwardedHeader(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/stop",
	)

	addMinerControlHeaderToken(req)

	req.Header.Set(
		"Forwarded",
		"for=203.0.113.10;proto=https",
	)

	rec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/stop",
	) {
		t.Fatal(
			"Forwarded proxy request was accepted",
		)
	}

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusForbidden,
		)
	}
}

func TestMinerControlSecurityRejectsProxyWithPublicHost(
	t *testing.T,
) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/start",
	)

	addMinerControlHeaderToken(req)

	// Backend connection is loopback, but Host identifies a public-facing
	// reverse proxy path. Loopback RemoteAddr alone must not grant trust.
	req.Host = "wallet.example:443"

	rec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		rec,
		req,
		"/api/miner/start",
	) {
		t.Fatal(
			"loopback proxy request with public Host was accepted",
		)
	}

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusForbidden,
		)
	}
}
