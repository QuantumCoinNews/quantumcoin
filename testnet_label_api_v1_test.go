package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	qcconfig "quantumcoin/config"
)

func TestNetworkLabelEndpointShowsTestnetV1(t *testing.T) {
	t.Setenv("QC_TESTNET", "1")

	oldCfg := cfg
	cfg = &qcconfig.Config{ChainFile: qcconfig.QCTestnetChainFileV1}
	defer func() { cfg = oldCfg }()

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
