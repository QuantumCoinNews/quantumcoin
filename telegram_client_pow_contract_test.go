package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"quantumcoin/wallet"
	"testing"
	"time"
)

// Explicitly invoked by the isolated Python/JavaScript client contract test.
func TestTelegramClientPoWContractV1(t *testing.T) {
	output := os.Getenv("QC_TEST_CLIENT_POW_CONTRACT")
	if output == "" {
		t.Skip("external test controller not requested")
	}
	h := unifiedSoloMux(t)
	chain, _ := nodeRuntime.Snapshot()
	initial := chain.BestHeightSnapshot()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/mine" {
			t.Error("Telegram asked the node to mine instead of submitting client PoW")
			http.Error(w, "server-side mining forbidden in this contract", 500)
			return
		}
		h.ServeHTTP(w, r)
	}))
	defer server.Close()
	data, err := json.Marshal(map[string]any{"baseURL": server.URL, "apiToken": actualMuxTestTokenV1, "address": wallet.NewWallet().GetAddress()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, data, 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(120 * time.Second)
	for {
		if _, err := os.Stat(output + ".done"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("client controller did not complete")
		}
		time.Sleep(25 * time.Millisecond)
	}
	if got := chain.BestHeightSnapshot(); got != initial+1 {
		t.Fatalf("got height %d, expected exactly one client block after %d", got, initial)
	}
}
