package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"quantumcoin/blockchain"
	qnode "quantumcoin/node"
)

const corsAcceptanceOriginV1 = "https://wallet.example"

func TestCORSRequestMetadataCannotReachSideEffectHandler(t *testing.T) {
	clearCORSTestEnv(t)
	t.Setenv("QC_CORS_ALLOWED_ORIGINS", corsAcceptanceOriginV1)
	for _, bind := range []string{"0.0.0.0:8082", "127.0.0.1:8082"} {
		t.Run(bind, func(t *testing.T) {
			if strings.HasPrefix(bind, "127.") {
				t.Setenv("QC_PUBLIC_TESTNET", "1")
			}
			for _, test := range []struct {
				name    string
				headers http.Header
			}{
				{"unknown", http.Header{"Origin": {"https://untrusted.example"}}},
				{"blank", http.Header{"Origin": {""}}},
				{"whitespace", http.Header{"Origin": {"   "}}},
				{"duplicate", http.Header{"Origin": {corsAcceptanceOriginV1, corsAcceptanceOriginV1}}},
				{"trusted-first", http.Header{"Origin": {corsAcceptanceOriginV1, "https://untrusted.example"}}},
				{"trusted-empty", http.Header{"Origin": {corsAcceptanceOriginV1, ""}}},
				{"case-alias", http.Header{"Origin": {corsAcceptanceOriginV1}, "origin": {"https://untrusted.example"}}},
				{"joined", http.Header{"Origin": {corsAcceptanceOriginV1 + ", https://untrusted.example"}}},
				{"opaque", http.Header{"Origin": {"null"}}},
			} {
				for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodOptions} {
					t.Run(test.name+"/"+method, func(t *testing.T) {
						mutations := 0
						handler := withCORSForListenV1(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { mutations++; w.WriteHeader(http.StatusAccepted) }), bind)
						request := httptest.NewRequest(method, "/api/test-mutation", strings.NewReader("simple form body"))
						request.Header = test.headers.Clone()
						request.Header.Set("Content-Type", "text/plain")
						response := httptest.NewRecorder()
						response.Header().Add("Vary", "Accept-Encoding")
						handler.ServeHTTP(response, request)
						if response.Code != http.StatusForbidden || mutations != 0 || response.Header().Get("Access-Control-Allow-Origin") != "" {
							t.Fatalf("origin boundary bypassed: status=%d mutations=%d", response.Code, mutations)
						}
						vary := strings.Join(response.Header().Values("Vary"), ",")
						if !strings.Contains(vary, "Accept-Encoding") || !strings.Contains(vary, "Origin") {
							t.Fatal("CORS erased the existing Vary contract", vary)
						}
					})
				}
			}
		})
	}
}

func corsSignedRequestFixtureV1(t *testing.T) (http.Handler, *blockchain.Blockchain, string, string) {
	t.Helper()
	chain, owner, to, config := networkSigningFixture(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	t.Setenv("QC_CORS_ALLOWED_ORIGINS", corsAcceptanceOriginV1)
	previousRuntime, previousConfig, previousMode := nodeRuntime, cfg, apiReadOnlyMode
	nodeRuntime, cfg, apiReadOnlyMode = qnode.NewRuntime(chain, config), config, false
	t.Cleanup(func() { nodeRuntime, cfg, apiReadOnlyMode = previousRuntime, previousConfig, previousMode })
	tx, err := signWalletIntentV1(chain, owner.GetAddress(), to.GetAddress(), "1", owner.ExportPrivateKeyHex(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !chain.VerifyTransaction(tx) {
		t.Fatal("fixture must carry a valid network-bound signature")
	}
	if err := chain.SaveToFile(config.ChainFile); err != nil {
		t.Fatal(err)
	}
	return newHTTPAPIHandlerForListenV1(nil, NewEndpointLimiter(), "0.0.0.0:8082"), chain, signedEnvelope(t, tx), config.ChainFile
}

func corsSignedRequestV1(handler http.Handler, method, body, token string, origins []string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/api/tx/send", strings.NewReader(body))
	request.RemoteAddr = "203.0.113.27:45678"
	request.Host = "node.example:8082"
	request.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	if origins != nil {
		request.Header["Origin"] = origins
	}
	if token != "" {
		request.Header.Set("X-QC-API-KEY", token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestCORSSignedTransferDeniedOriginsPreserveLedger(t *testing.T) {
	handler, chain, envelope, file := corsSignedRequestFixtureV1(t)
	before := chain.StateSnapshot()
	beforeDisk, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		origins []string
	}{
		{"unknown", []string{"https://untrusted.example"}},
		{"scheme", []string{"http://wallet.example"}},
		{"suffix", []string{"https://wallet.example.untrusted.example"}},
		{"port", []string{"https://wallet.example:444"}},
		{"opaque", []string{"null"}},
		{"blank", []string{""}},
		{"trusted-first", []string{corsAcceptanceOriginV1, "https://untrusted.example"}},
		{"duplicate", []string{corsAcceptanceOriginV1, corsAcceptanceOriginV1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := corsSignedRequestV1(handler, http.MethodPost, envelope, actualMuxTestTokenV1, test.origins)
			if response.Code != http.StatusForbidden || response.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatalf("signed mutation was not CORS-rejected: status=%d", response.Code)
			}
			if !reflect.DeepEqual(before, chain.StateSnapshot()) {
				t.Fatal("denied origin changed authoritative chain or mempool")
			}
		})
	}
	afterDisk, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(beforeDisk, afterDisk) {
		t.Fatal("denied requests changed the chain file", err)
	}
	response := corsSignedRequestV1(handler, http.MethodPost, envelope, actualMuxTestTokenV1, []string{corsAcceptanceOriginV1})
	if response.Code != http.StatusOK || chain.PendingTransactionCountSnapshot() != 1 {
		t.Fatalf("positive control failed after denied requests: status=%d", response.Code)
	}
}

func TestCORSNativeAndAllowedBrowserStillRequireAuthentication(t *testing.T) {
	for _, origins := range [][]string{nil, {corsAcceptanceOriginV1}} {
		t.Run(map[bool]string{true: "native", false: "browser"}[origins == nil], func(t *testing.T) {
			handler, chain, envelope, _ := corsSignedRequestFixtureV1(t)
			if response := corsSignedRequestV1(handler, http.MethodPost, envelope, "wrong-test-token", origins); response.Code != http.StatusUnauthorized || chain.PendingTransactionCountSnapshot() != 0 {
				t.Fatal("CORS replaced route authentication", response.Code)
			}
			if response := corsSignedRequestV1(handler, http.MethodPost, envelope, actualMuxTestTokenV1, origins); response.Code != http.StatusOK || chain.PendingTransactionCountSnapshot() != 1 {
				t.Fatal("legitimate signed client failed", response.Code)
			}
		})
	}
}

func TestCORSWireEmptyAndRepeatedOriginsFailClosed(t *testing.T) {
	clearCORSTestEnv(t)
	t.Setenv("QC_CORS_ALLOWED_ORIGINS", corsAcceptanceOriginV1)
	var mutations atomic.Int32
	server := httptest.NewServer(withCORSForListenV1(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { mutations.Add(1); w.WriteHeader(http.StatusAccepted) }), "0.0.0.0:8082"))
	defer server.Close()
	for _, origins := range [][]string{{""}, {corsAcceptanceOriginV1, "https://untrusted.example"}} {
		request, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("simple body"))
		if err != nil {
			t.Fatal(err)
		}
		request.Header["Origin"] = origins
		request.Header.Set("Content-Type", "text/plain")
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if err != nil || response.StatusCode != http.StatusForbidden || mutations.Load() != 0 {
			t.Fatalf("wire origin bypass: status=%d mutations=%d error=%v", response.StatusCode, mutations.Load(), err)
		}
	}
}
