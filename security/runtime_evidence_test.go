package security

import "testing"

func TestRuntimeEvidenceInvokesAllActiveAnalyses(t *testing.T) {
	evidence := &RuntimeEvidence{}
	for i := 0; i < 30; i++ {
		evidence.ObserveHTTP("127.0.0.1", "/api/tx/send", "POST", 429)
	}
	evidence.ObserveSpend("tx", "wallet", "input already pending in mempool")
	evidence.ObserveMiner("MINER_FAILED")
	out := evidence.Reports(true, true)
	for _, key := range []string{"actionBurstGuard", "ipApiReputation", "guiCliApiGuardrails", "minerSafety", "doubleSpendConflict"} {
		if out[key] == nil || out[key]["overallRisk"] == nil {
			t.Fatal("missing analysis", key)
		}
	}
	if out["actionBurstGuard"]["overallRisk"] == "LOW" || out["doubleSpendConflict"]["overallRisk"] == "LOW" {
		t.Fatal("observed rejection ignored")
	}
	for i := 0; i < 2100; i++ {
		evidence.ObserveHTTP("local", "/api/health", "GET", 200)
	}
	if len(evidence.http) > 2000 {
		t.Fatal("HTTP evidence unbounded")
	}
	unknown := EvaluateNodeSyncDrift(NodeSyncDriftInput{LocalHeight: 100, PeerCount: 2, NetworkHeightUnknown: true, LatestBlockAgeSeconds: 1})
	if unknown["health"] == "ISOLATED" || unknown["networkHeight"] != nil || unknown["risk"] != "UNKNOWN" {
		t.Fatal("unknown network height fabricated", unknown)
	}
}
