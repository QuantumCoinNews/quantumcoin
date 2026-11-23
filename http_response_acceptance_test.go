package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPResponseMuxErrorSchema(t *testing.T) {
	externalMiningMux(t)
	handler := newHTTPAPIHandlerV1(NewHTTPGuard(), NewEndpointLimiter())
	for _, row := range []struct {
		method, path, body string
		auth, readonly     bool
		status             int
	}{
		{"GET", "/api/unknown", "", true, false, 404},
		{"GET", "/api/wallet/address", "", false, false, 401},
		{"POST", "/api/wallet/address", "", true, false, 405},
		{"GET", "/api/wallet/balance/not-a-wallet", "", true, false, 400},
		{"GET", "/api/stake/status", "", true, false, 501},
		{"GET", "/api/nft/monthly/mint", "", true, false, 405},
		{"POST", "/api/mine", "{}", true, true, 403},
		{"POST", "/api/tx/send", strings.Repeat("x", int(apiMaxBodyBytes)+1), true, false, 413},
	} {
		t.Run(row.path+"/"+row.method, func(t *testing.T) {
			nodeRuntime.SetAPIReadOnly(row.readonly)
			defer nodeRuntime.SetAPIReadOnly(false)
			req := httptest.NewRequest(row.method, row.path, strings.NewReader(row.body))
			req.RemoteAddr = "127.0.0.1:43035"
			req.Host = "127.0.0.1:8082"
			if row.auth {
				req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assertRoutingJSONErrorV1(t, w.Code, w.Header(), w.Body.Bytes(), row.status)
		})
	}
}

func TestHTTPResponseEncodingFailureOverTCP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"unsupported": make(chan int)})
	}))
	defer server.Close()
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	assertRoutingJSONErrorV1(t, response.StatusCode, response.Header, body, 500)
}

func TestHTTPResponseSuccessfulDTOsAreNotRewrapped(t *testing.T) {
	for _, value := range []any{WalletResponse{Address: "public-address"}, []any{}, map[string]any{"success": true, "txid": "public-id"}} {
		w := httptest.NewRecorder()
		writeOK(w, value)
		expected, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || strings.TrimSpace(w.Body.String()) != string(expected) {
			t.Fatal("success contract changed", w.Code, w.Body.String())
		}
	}
}
