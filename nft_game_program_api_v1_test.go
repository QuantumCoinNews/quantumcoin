package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	qnode "quantumcoin/node"
)

func TestNFTGameBridgeMuxProtectsCanonicalRouteV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	old := nodeRuntime
	nodeRuntime = nil
	t.Cleanup(func() { nodeRuntime = old })
	h := newActualMuxTestHandlerV1(t)
	for _, target := range []string{"/api/nft/monthly/program", "/api/nft/monthly/program?token=" + actualMuxTestTokenV1} {
		req := newActualMuxRequestV1(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("canonical route auth missing: %d %s", rec.Code, rec.Body)
		}
	}
	req := newActualMuxRequestV1(http.MethodGet, "/api/nft/monthly/program", nil)
	addActualMuxTokenV1(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Fatal("QMA0066_GAME_BRIDGE_RED: canonical protected program route is missing")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("absent runtime returned %d %s", rec.Code, rec.Body)
	}
}

func TestNFTGameBridgeMuxReadsDisabledRuntimeWithoutSideEffectsV1(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	old := nodeRuntime
	chain := blockchain.NewBlockchain(blockchain.CanonicalInitialMiningRewardV1, int(blockchain.QCTotalSupplyHardCap))
	nodeRuntime = qnode.NewRuntime(chain, config.Default())
	t.Cleanup(func() { nodeRuntime = old })
	// Gob map iteration is not deterministic; compare detached state, rather
	// than interpreting a different serialization order as a chain mutation.
	before := chain.StateSnapshot()
	activationBefore := chain.ActivationPolicySnapshotV1()
	h := newActualMuxTestHandlerV1(t)
	req := newActualMuxRequestV1(http.MethodGet, "/api/nft/monthly/program", nil)
	addActualMuxTokenV1(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var p map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || p["success"] != true || p["verified"] != false || p["gameActivatedAt"] != float64(0) || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("disabled canonical runtime: %d %s", rec.Code, rec.Body)
	}
	if !reflect.DeepEqual(before, chain.StateSnapshot()) || !reflect.DeepEqual(activationBefore, chain.ActivationPolicySnapshotV1()) {
		t.Fatal("program read mutated chain")
	}
	req = newActualMuxRequestV1(http.MethodPost, "/api/nft/monthly/program", nil)
	addActualMuxTokenV1(req)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("program POST: %d %s", rec.Code, rec.Body)
	}
}
