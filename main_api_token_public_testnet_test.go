package main

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func clearAPITokenTestEnv(t *testing.T) {
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

func TestCheckAPITokenEveryModeRequiresConfiguredToken(t *testing.T) {
	clearAPITokenTestEnv(t)

	req := httptest.NewRequest("GET", "/api/wallet/address", nil)
	rr := httptest.NewRecorder()

	if checkAPIToken(rr, req) {
		t.Fatal("expected every mode to reject missing QC_API_TOKEN")
	}
	if rr.Code != 401 {
		t.Fatalf("expected 401 for missing token configuration, got %d", rr.Code)
	}
}
func TestCheckAPITokenPublicTestnetRequiresConfiguredToken(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_PUBLIC_TESTNET", "1")

	req := httptest.NewRequest("GET", "/api/wallet/address", nil)
	rr := httptest.NewRecorder()

	if checkAPIToken(rr, req) {
		t.Fatalf("expected public testnet mode to reject missing QC_API_TOKEN")
	}

	if rr.Code != 401 {
		t.Fatalf("expected 401 for missing public testnet token config, got %d", rr.Code)
	}
}

func TestCheckAPITokenPublicTestnetAcceptsHeaderToken(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_PUBLIC_TESTNET", "1")
	t.Setenv("QC_API_TOKEN", "secret-token")

	req := httptest.NewRequest("GET", "/api/wallet/address", nil)
	req.Header.Set("X-QC-API-KEY", "secret-token")
	rr := httptest.NewRecorder()

	if !checkAPIToken(rr, req) {
		t.Fatalf("expected matching X-QC-API-KEY to pass in public testnet mode")
	}
}

func TestCheckAPITokenPublicTestnetAcceptsBearerToken(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_PUBLIC_TESTNET", "1")
	t.Setenv("QC_API_TOKEN", "secret-token")

	req := httptest.NewRequest("GET", "/api/wallet/address", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()

	if !checkAPIToken(rr, req) {
		t.Fatalf("expected matching Authorization bearer token to pass in public testnet mode")
	}
}

func TestCheckAPITokenPublicListenTestnetRequiresToken(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_NET_MAGIC", "QC_TESTNET_V1")
	t.Setenv("QC_API_LISTEN", "0.0.0.0:8082")

	req := httptest.NewRequest("GET", "/api/wallet/address", nil)
	rr := httptest.NewRecorder()

	if checkAPIToken(rr, req) {
		t.Fatalf("expected publicly exposed testnet API to reject missing QC_API_TOKEN")
	}
}

func TestCheckAPITokenEveryModeRejectsQueryToken(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_API_TOKEN", "secret-token")

	req := httptest.NewRequest(
		"GET",
		"/api/wallet/address?token=secret-token",
		nil,
	)
	rr := httptest.NewRecorder()

	if checkAPIToken(rr, req) {
		t.Fatal("expected every mode to reject query token transport")
	}
	if rr.Code != 401 {
		t.Fatalf("expected 401 for query token transport, got %d", rr.Code)
	}
}

func TestCheckAPITokenEveryModeRejectsFormToken(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_API_TOKEN", "secret-token")

	req := httptest.NewRequest(
		"POST",
		"/api/wallet/address",
		strings.NewReader("token=secret-token"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	if checkAPIToken(rr, req) {
		t.Fatal("expected every mode to reject form token transport")
	}
	if rr.Code != 401 {
		t.Fatalf("expected 401 for form token transport, got %d", rr.Code)
	}
}
func TestCheckAPITokenPublicTestnetRejectsQueryToken(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_PUBLIC_TESTNET", "1")
	t.Setenv("QC_API_TOKEN", "secret-token")

	req := httptest.NewRequest("GET", "/api/wallet/address?token=secret-token", nil)
	rr := httptest.NewRecorder()

	if checkAPIToken(rr, req) {
		t.Fatalf("expected public testnet mode to reject query token fallback")
	}
}

func TestCheckAPITokenPublicTestnetRejectsFormToken(t *testing.T) {
	clearAPITokenTestEnv(t)
	t.Setenv("QC_PUBLIC_TESTNET", "1")
	t.Setenv("QC_API_TOKEN", "secret-token")

	req := httptest.NewRequest("POST", "/api/wallet/address", strings.NewReader("token=secret-token"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	if checkAPIToken(rr, req) {
		t.Fatalf("expected public testnet mode to reject form token fallback")
	}
}
