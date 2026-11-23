package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	qnode "quantumcoin/node"
	"testing"
)

func TestRuntimePolicyAPIRequiresAuthAndObservesChain(t *testing.T) {
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	previous := nodeRuntime
	t.Cleanup(func() { nodeRuntime = previous })
	chain := blockchain.NewBlockchain(50, int(blockchain.QCTotalSupplyHardCap))
	configuration := config.Default()
	nodeRuntime = qnode.NewRuntime(chain, configuration)
	handler := newActualMuxTestHandlerV1(t)
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, newActualMuxRequestV1(http.MethodGet, "/api/runtime/policy", nil))
	if missing.Code != 401 {
		t.Fatalf("unauthed status=%d", missing.Code)
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		request := newActualMuxRequestV1(method, "/api/runtime/policy", nil)
		addActualMuxTokenV1(request)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if method == http.MethodPost {
			if recorder.Code != 405 {
				t.Fatalf("write status=%d", recorder.Code)
			}
			continue
		}
		if recorder.Code != 200 {
			t.Fatalf("get status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		var body struct {
			Version  string
			Source   string
			Context  map[string]any
			Sections map[string]any
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Version != "NODE_RUNTIME_POLICY_V1" || body.Source != "quantumcoin_go_node" || body.Context["atomsPerQc"] != "1000000" || body.Context["nextProtocol"] != "QC_LEGACY_PROTOCOL_V1" {
			t.Fatalf("policy=%s", recorder.Body.String())
		}
		for _, section := range []string{"mainnet-before", "burn-sink", "network-fee", "founder-vesting", "hardfork-policy", "mining-transition"} {
			if body.Sections[section] == nil {
				t.Fatalf("missing %s", section)
			}
		}
	}
	if chain.BestHeightSnapshot() != 0 {
		t.Fatal("policy read mutated chain")
	}
	nodeRuntime = nil
	request := newActualMuxRequestV1(http.MethodGet, "/api/runtime/policy", nil)
	addActualMuxTokenV1(request)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != 503 {
		t.Fatalf("offline status=%d", recorder.Code)
	}
}
