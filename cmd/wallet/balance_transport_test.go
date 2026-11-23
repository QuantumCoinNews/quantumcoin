package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"quantumcoin/wallet"
)

func TestWalletReadsCanonicalBalanceOnly(t *testing.T) {
	address := wallet.NewWallet().GetAddress()
	t.Setenv("QC_API_TOKEN", "wallet-balance-fixture")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/api/wallet/balance/"+address || r.Method != http.MethodGet ||
			r.Header.Get("X-QC-API-KEY") != "wallet-balance-fixture" {
			http.Error(w, "unexpected request", 400)
			return
		}
		fmt.Fprint(w, `{"balanceQc":"0.000003","spendableQc":"0.000001"}`)
	}))
	defer server.Close()
	origin, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	previous := apiPort
	t.Cleanup(func() { apiPort = previous })
	apiPort = origin.Port()
	spendable, locked, ok := getBalanceUniversal(address)
	if !ok || spendable != 0.000001 || locked != 0.000002 || calls.Load() != 1 {
		t.Fatalf("unexpected canonical wallet result: %f %f %t calls=%d", spendable, locked, ok, calls.Load())
	}
}
