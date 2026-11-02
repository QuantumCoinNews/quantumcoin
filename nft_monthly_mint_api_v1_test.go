package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	qnode "quantumcoin/node"
)

func TestNFTMintTransportMuxProtectsMintAndReceiptRoutesV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	old := nodeRuntime
	nodeRuntime = nil
	t.Cleanup(func() { nodeRuntime = old })
	h := newActualMuxTestHandlerV1(t)
	for _, route := range []string{"/api/nft/monthly/mint", "/api/nft/monthly/receipt"} {
		req := newActualMuxRequestV1(http.MethodPost, route, nil)
		addActualMuxTokenV1(req)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatal("QMA0066_MINT_HTTP_RED: protected mint/receipt transport is missing")
		}
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("absent runtime route %s: %d %s", route, rec.Code, rec.Body)
		}
		for _, target := range []string{route, route + "?token=" + actualMuxTestTokenV1} {
			req := newActualMuxRequestV1(http.MethodPost, target, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("unsigned route %s returned %d %s", target, rec.Code, rec.Body)
			}
		}
	}
}

func TestNFTMintTransportMuxRejectsDisabledAndReadOnlyRuntimeV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	old := nodeRuntime
	cfg := config.Default()
	cfg.ChainFile = filepath.Join(t.TempDir(), "unwritten-chain.dat")
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	nodeRuntime = qnode.NewRuntime(chain, cfg)
	t.Cleanup(func() { nodeRuntime = old })
	before := chain.StateSnapshot()
	// Syntactically valid public evidence. An unverified runtime must not
	// install the request's scope/signers as its local activation or authority.
	body := `{"version":"QC_NFT_MONTHLY_MINT_REQUEST_V1","record":{"networkID":"test-only-disabled","campaignID":"telegram-monthly","awardPeriod":"2030-01","serial":1,"recipientPubKeyHash":"` + strings.Repeat("11", 20) + `","metadataHash":"` + strings.Repeat("22", 32) + `"},"decision":{"networkID":"test-only-disabled","legacyGenesisHex":"` + strings.Repeat("33", 32) + `","recordDigest":"` + strings.Repeat("44", 32) + `"},"approvals":[{"publicKey":"04` + strings.Repeat("55", 64) + `","signature":"01010101"}]}`
	h := newActualMuxTestHandlerV1(t)
	for _, route := range []string{"/api/nft/monthly/mint", "/api/nft/monthly/receipt"} {
		req := newActualMuxRequestV1(http.MethodPost, route, bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		addActualMuxTokenV1(req)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("unverified route %s returned %d %s", route, rec.Code, rec.Body)
		}
	}
	nodeRuntime.SetAPIReadOnly(true)
	req := newActualMuxRequestV1(http.MethodPost, "/api/nft/monthly/mint", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	addActualMuxTokenV1(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !reflect.DeepEqual(before, chain.StateSnapshot()) {
		t.Fatalf("read-only/disabled transport changed state or accepted mint: %d %s", rec.Code, rec.Body)
	}
}
