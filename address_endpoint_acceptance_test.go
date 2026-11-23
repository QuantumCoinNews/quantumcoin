package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	qnode "quantumcoin/node"
	"quantumcoin/wallet"
)

func TestAddressEndpointRegressionWhitespacePath(t *testing.T) {
	handler := externalMiningMux(t)
	address := wallet.NewWallet().GetAddress()
	for _, value := range []string{" " + address, address + " ", "\t" + address} {
		w := webRequest(handler, "GET", "/api/wallet/balance/"+url.PathEscape(value), "")
		assertRoutingJSONErrorV1(t, w.Code, w.Header(), w.Body.Bytes(), 400)
	}
}

func TestAddressEndpointMuxValidationAndAvailability(t *testing.T) {
	handler := externalMiningMux(t)
	chain, _ := nodeRuntime.Snapshot()
	address := wallet.NewWallet().GetAddress()
	broken := address[:len(address)-1] + "1"
	if broken == address {
		broken = address[:len(address)-1] + "2"
	}
	if wallet.ValidateAddress(broken) {
		t.Fatal("bad-checksum fixture unexpectedly valid")
	}
	before, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", " ", "@@@", "0OIl", broken, "1", strings.Repeat("z", 4096), "a/b", "\x00"} {
		t.Run(url.PathEscape(value)[:min(len(url.PathEscape(value)), 48)], func(t *testing.T) {
			w := webRequest(handler, "GET", "/api/wallet/balance/"+url.PathEscape(value), "")
			assertRoutingJSONErrorV1(t, w.Code, w.Header(), w.Body.Bytes(), 400)
		})
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		w := webRequest(handler, method, "/api/wallet/balance/"+address, "")
		assertRoutingJSONErrorV1(t, w.Code, w.Header(), w.Body.Bytes(), 405)
		if w.Header().Get("Allow") != "GET" {
			t.Fatal("missing Allow: GET")
		}
	}
	req := httptest.NewRequest("GET", "/api/wallet/balance/"+address, nil)
	req.RemoteAddr = "127.0.0.1:43033"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), 401)
	for _, readonly := range []bool{false, true} {
		nodeRuntime.SetAPIReadOnly(readonly)
		response = webRequest(handler, "GET", "/api/wallet/balance/"+address, "")
		var balance BalanceResponse
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &balance) != nil || balance.BalanceAtoms != "0" || balance.SpendableAtoms != "0" || balance.Height != 0 {
			t.Fatalf("invalid zero balance: %d %s", response.Code, response.Body.String())
		}
	}
	nodeRuntime.SetAPIReadOnly(false)
	saved := nodeRuntime
	for _, missingRuntime := range []bool{false, true} {
		nodeRuntime = qnode.NewRuntime(nil, cfg)
		if missingRuntime {
			nodeRuntime = nil
		}
		h := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
		response = webRequest(h, "GET", "/api/wallet/balance/"+address, "")
		nodeRuntime = saved
		assertRoutingJSONErrorV1(t, response.Code, response.Header(), response.Body.Bytes(), 503)
	}
	after, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || len(chain.PendingTransactionsSnapshot()) != 0 || chain.BestHeightSnapshot() != 0 {
		t.Fatal("address request changed ledger")
	}
}

func TestAddressEndpointMissingOrInvalidConfiguredWallet(t *testing.T) {
	handler := externalMiningMux(t)
	previousDir := activeCLIPathsV1.DataDir
	activeCLIPathsV1.DataDir = t.TempDir()
	defer func() { activeCLIPathsV1.DataDir = previousDir }()
	t.Setenv("APPDATA", t.TempDir())
	for _, address := range []string{"", "not-a-wallet", strings.Repeat("z", 4096)} {
		cfg.Miner.Address = address
		w := webRequest(handler, "GET", "/api/wallet/address", "")
		assertRoutingJSONErrorV1(t, w.Code, w.Header(), w.Body.Bytes(), 503)
	}
	entries, err := os.ReadDir(activeCLIPathsV1.DataDir)
	if err != nil || len(entries) != 0 {
		t.Fatal("read endpoint created wallet or address files", err)
	}
	cfg.Miner.Address = wallet.NewWallet().GetAddress()
	w := webRequest(handler, "GET", "/api/wallet/address", "")
	var result WalletResponse
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Address != cfg.Miner.Address {
		t.Fatal("configured wallet unavailable", w.Code, w.Body.String())
	}
}
