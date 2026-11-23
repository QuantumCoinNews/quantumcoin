package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	qcconfig "quantumcoin/config"
	qnode "quantumcoin/node"
)

func TestNetworkLabelEndpointShowsTestnetV1(t *testing.T) {
	oldRuntime := nodeRuntime
	t.Cleanup(func() { nodeRuntime = oldRuntime })
	chain := blockchain.NewBlockchain(50, 1000)
	if err := chain.BindRuntimeNetworkIdentityV1("testnet-local"); err != nil {
		t.Fatal(err)
	}
	nodeRuntime = qnode.NewRuntime(chain, &qcconfig.Config{ChainFile: qcconfig.QCTestnetChainFileV1})
	// Late environment changes cannot relabel the live runtime.
	t.Setenv("QC_MAINNET", "1")

	req := httptest.NewRequest(http.MethodGet, "/api/network/label", nil)
	rr := httptest.NewRecorder()

	handleNetworkLabelV1(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("X-QuantumCoin-Network-Label"); got != "TESTNET" {
		t.Fatalf("expected TESTNET header, got %q", got)
	}
	if got := rr.Header().Get("X-QuantumCoin-Testnet"); got != "true" {
		t.Fatalf("expected testnet header true, got %q", got)
	}
	if !strings.Contains(rr.Body.String(), `"label":"TESTNET"`) {
		t.Fatalf("expected TESTNET body, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"is_testnet":true`) {
		t.Fatalf("expected is_testnet true, got %s", rr.Body.String())
	}
}

func TestNetworkLabelEndpointWithoutIdentityIsUnverifiedV1(t *testing.T) {
	oldRuntime := nodeRuntime
	t.Cleanup(func() { nodeRuntime = oldRuntime })
	t.Setenv("QC_MAINNET", "1")
	nodeRuntime = nil
	w := httptest.NewRecorder()
	handleNetworkLabelV1(w, httptest.NewRequest(http.MethodGet, "/api/network/label", nil))
	if w.Code != http.StatusOK || w.Header().Get("X-QuantumCoin-Network-Label") != "UNVERIFIED" {
		t.Fatalf("unbound node received a network label: status=%d body=%s", w.Code, w.Body.String())
	}
}
