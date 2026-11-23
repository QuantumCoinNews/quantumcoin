package main

import (
	"bytes"
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

func assertRoutingJSONErrorV1(t *testing.T, status int, header http.Header, body []byte, want int) {
	t.Helper()
	var payload struct {
		Success *bool  `json:"success"`
		Message string `json:"message"`
	}
	if status != want || !strings.Contains(header.Get("Content-Type"), "application/json") || json.Unmarshal(body, &payload) != nil || payload.Success == nil || *payload.Success || payload.Message == "" {
		t.Fatalf("status=%d want=%d headers=%v body=%q", status, want, header, body)
	}
}

func TestAPIRoutingNonCanonicalPathsDoNotRedirect(t *testing.T) {
	handler := externalMiningMux(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, target := range []string{"/api/../wallet", "/api//missing", "/page/../api/unknown", "/api/./missing", "/api/%2e%2e/wallet", "/api%2f..%2fwallet"} {
		t.Run(target, func(t *testing.T) {
			req, err := http.NewRequest("GET", server.URL+target, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			assertRoutingJSONErrorV1(t, response.StatusCode, response.Header, body, 404)
			if response.Header.Get("Location") != "" {
				t.Fatal("API path redirected", response.Header)
			}
		})
	}
}

func TestAPIRoutingUnknownPathsAndMethodBoundaries(t *testing.T) {
	handler := externalMiningMux(t)
	chain, _ := nodeRuntime.Snapshot()
	before, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/api", "/api/", "/api/unknown", "/api/unknown/nested", "/a%70i/unknown", "/api%2funknown", "/api/wallet/address/missing", "/api/ai/analysis/missing", "/api/miner/status/missing", "/api/not-a-route.js"} {
		for _, method := range []string{"GET", "POST", "DELETE"} {
			t.Run(method+target, func(t *testing.T) {
				response := webRequest(handler, method, target, `{}`)
				assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), 404)
			})
		}
	}
	for _, token := range []string{"", actualMuxTestTokenV1} {
		nodeRuntime.SetAPIReadOnly(true)
		req := httptest.NewRequest("GET", "/api/unknown", nil)
		req.RemoteAddr = "127.0.0.1:43332"
		req.Header.Set("X-QC-API-KEY", token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		want := 403
		if token == "" {
			want = 401
		}
		assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), want)
		nodeRuntime.SetAPIReadOnly(false)
	}
	after, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || chain.BestHeightSnapshot() != 0 || len(chain.PendingTransactionsSnapshot()) != 0 || nodeRuntime.IsMining() {
		t.Fatal("unknown route changed chain/miner state")
	}
}

func TestAPIRoutingRequiredReadEndpoints(t *testing.T) {
	handler := externalMiningMux(t)
	address := wallet.NewWallet().GetAddress()
	cfg.Miner.Address = address
	t.Setenv("QC_API_LISTEN", "127.0.0.1:8082")
	for _, row := range []struct {
		path   string
		public bool
		fields []string
		array  bool
	}{
		{"/api/wallet/address", false, []string{"address"}, false},
		{"/api/network/label", true, []string{"network", "label"}, false},
		{"/api/ai/analysis", false, []string{"anomaly_report", "recommendations", "reward_suggestions"}, false},
		{"/api/ai/bonus", false, nil, true},
		{"/api/miner/status", false, []string{"running", "height", "bits", "httpPort"}, false},
	} {
		t.Run(row.path, func(t *testing.T) {
			for _, readonly := range []bool{false, true} {
				nodeRuntime.SetAPIReadOnly(readonly)
				req := httptest.NewRequest("GET", row.path, nil)
				req.RemoteAddr = "127.0.0.1:43332"
				req.Host = "127.0.0.1:8082"
				if !row.public {
					req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, req)
				if response.Code != 200 || !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
					t.Fatalf("read-only=%v status=%d body=%s", readonly, response.Code, response.Body.String())
				}
				if row.array {
					var values []json.RawMessage
					if json.Unmarshal(response.Body.Bytes(), &values) != nil {
						t.Fatalf("bonus history should be a list or null: %s", response.Body.String())
					}
				} else {
					var values map[string]json.RawMessage
					if json.Unmarshal(response.Body.Bytes(), &values) != nil {
						t.Fatal("not a JSON object")
					}
					for _, field := range row.fields {
						if _, ok := values[field]; !ok {
							t.Errorf("missing field %s: %s", field, response.Body.String())
						}
					}
					if row.path == "/api/wallet/address" && string(values["address"]) != `"`+address+`"` {
						t.Fatal("address differs from configured wallet")
					}
				}
			}
			nodeRuntime.SetAPIReadOnly(false)
			req := httptest.NewRequest("POST", row.path, nil)
			req.RemoteAddr = "127.0.0.1:43332"
			req.Host = "127.0.0.1:8082"
			req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), 405)
			if !row.public {
				req = httptest.NewRequest("GET", row.path, nil)
				req.RemoteAddr = "127.0.0.1:43332"
				response = httptest.NewRecorder()
				handler.ServeHTTP(response, req)
				assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), 401)
			}
		})
	}
}
