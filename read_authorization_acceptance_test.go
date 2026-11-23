package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"quantumcoin/wallet"
)

func TestReadAuthorizationRegressionQuota(t *testing.T) {
	externalMiningMux(t)
	for _, path := range []string{"/api/ai/analysis", "/api/security/dashboard"} {
		t.Run(path, func(t *testing.T) {
			h := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
			for range 4 {
				response := readOnlyRequestV1(h, "GET", path, "incorrect-fixture-token")
				if response.Code != 401 {
					t.Fatalf("unauthenticated response=%d", response.Code)
				}
			}
			response := readOnlyRequestV1(h, "GET", path, actualMuxTestTokenV1)
			if response.Code != 200 {
				t.Fatalf("anonymous traffic consumed authenticated read quota: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestReadAuthorizationRegressionNoStore(t *testing.T) {
	h := externalMiningMux(t)
	for _, token := range []string{"", actualMuxTestTokenV1} {
		response := readOnlyRequestV1(h, "GET", "/api/blocks", token)
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("private read/denial missing no-store: status=%d", response.Code)
		}
		vary := strings.Join(response.Header().Values("Vary"), ",")
		if !strings.Contains(vary, "Authorization") || !strings.Contains(vary, "X-QC-API-KEY") {
			t.Errorf("private credential cache key missing: %q", vary)
		}
	}
}

func TestReadAuthorizationRegressionPublicLabelPrivacy(t *testing.T) {
	h := externalMiningMux(t)
	t.Setenv("QC_API_TOKEN", "")
	response := readOnlyRequestV1(h, "GET", "/api/network/label", "")
	var info map[string]json.RawMessage
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &info) != nil {
		t.Fatal("public label unavailable", response.Code)
	}
	if _, exists := info["chain_file"]; exists || strings.Contains(response.Body.String(), cfg.ChainFile) {
		t.Fatalf("public label exposes local storage path: %s", response.Body.String())
	}
	if len(info["network"]) == 0 || len(info["label"]) == 0 {
		t.Fatal("public network identity lost")
	}
}

func TestReadAuthorizationEveryPrivateRead(t *testing.T) {
	externalMiningMux(t)
	paths := []string{
		"/api/monitoring", "/api/health", "/api/capabilities", "/api/telemetry", "/api/economy/policy", "/api/runtime/policy", "/api/nft/monthly/program",
		"/api/security/summary", "/api/security/events", "/api/security/score", "/api/security/node-health", "/api/security/peer-reputation",
		"/api/security/dashboard", "/api/security/admin-report", "/api/security/wallet-anomaly", "/api/security/mempool-abuse",
		"/api/security/anti-spam-fee-policy", "/api/security/finality-checkpoint", "/api/wallet/address", "/api/wallet/balance/" + wallet.NewWallet().GetAddress(),
		"/api/tx/status?id=not-a-private-id", "/api/ai/alerts", "/api/ai/bonus", "/api/ai/analysis", "/api/ai/security-dashboard",
		"/api/blocks", "/api/block?index=0", "/api/stake/status", "/api/mine/job", "/api/mine/tip", "/api/mine/result", "/api/miner/status",
	}
	for _, bind := range []string{"127.0.0.1:8082", "0.0.0.0:8082"} {
		t.Setenv("QC_API_LISTEN", bind)
		for _, readonly := range []bool{false, true} {
			nodeRuntime.SetAPIReadOnly(readonly)
			h := newHTTPAPIHandlerForListenV1(nil, NewEndpointLimiter(), bind)
			for _, path := range paths {
				for _, mode := range []string{"missing", "incorrect", "basic"} {
					t.Run(bind+"/"+map[bool]string{false: "writable", true: "read-only"}[readonly]+path+"/"+mode, func(t *testing.T) {
						req := httptest.NewRequest("GET", path, nil)
						req.RemoteAddr = "127.0.0.1:43036"
						req.Host = "127.0.0.1:8082"
						if mode == "incorrect" {
							req.Header.Set("X-QC-API-KEY", "incorrect-fixture-token")
						}
						if mode == "basic" {
							req.Header.Set("Authorization", "Basic not-a-bearer-token")
						}
						response := httptest.NewRecorder()
						h.ServeHTTP(response, req)
						assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), 401)
						if response.Header().Get("WWW-Authenticate") == "" {
							t.Fatal("missing authentication challenge")
						}
						var body map[string]any
						if json.Unmarshal(response.Body.Bytes(), &body) != nil || len(body) != 2 {
							t.Fatal("unauthorized response contains extra data", response.Body.String())
						}
					})
				}
			}
		}
	}
	nodeRuntime.SetAPIReadOnly(false)
	t.Setenv("QC_API_TOKEN", "")
	h := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	for _, path := range paths {
		response := readOnlyRequestV1(h, "GET", path, actualMuxTestTokenV1)
		assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), 401)
	}
}

func TestReadAuthorizationOverTCPAndRetiredDataRoutes(t *testing.T) {
	externalMiningMux(t)
	chain, _ := nodeRuntime.Snapshot()
	before, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	h := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	server := httptest.NewServer(h)
	defer server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	for _, bearer := range []bool{false, true} {
		req, err := http.NewRequest("GET", server.URL+"/api/blocks", nil)
		if err != nil {
			t.Fatal(err)
		}
		if bearer {
			req.Header.Set("Authorization", "Bearer "+actualMuxTestTokenV1)
		}
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !bearer {
			assertRoutingJSONErrorV1(t, response.StatusCode, response.Header, body, 401)
		} else {
			var blocks []json.RawMessage
			if response.StatusCode != 200 || json.Unmarshal(body, &blocks) != nil || len(blocks) != 1 {
				t.Fatal("authenticated read failed", response.StatusCode, string(body))
			}
		}
	}
	for _, path := range []string{"/api/mempool", "/api/utxo", "/api/address/balance?addr=anything", "/a%70i/mempool", "/api%2futxo"} {
		response := readOnlyRequestV1(h, "GET", path, "")
		assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), 404)
	}
	after, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) || chain.BestHeightSnapshot() != 0 || len(chain.PendingTransactionsSnapshot()) != 0 || nodeRuntime.IsMining() {
		t.Fatal("read authorization changed ledger/miner")
	}
}
