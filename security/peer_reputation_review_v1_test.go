package security

import (
	"os"
	"strings"
	"testing"
)

func TestPeerReputationReviewV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := AnalyzePeerReputation([]PeerReputationObservation{
		{
			PeerID: "peer-good",
			Score:  95,
			Status: "GOOD",
		},
	})

	if out["version"] != PeerReputationReviewVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], PeerReputationReviewVersion)
	}
	if out["mode"] != "READ_ONLY_P2P_REPUTATION_REVIEW" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["overallRisk"] != "LOW" {
		t.Fatalf("risk mismatch: got=%v out=%v", out["overallRisk"], out)
	}

	note, ok := out["policyNote"].(string)
	if !ok || note == "" {
		t.Fatalf("policyNote must exist")
	}
	if !strings.Contains(note, "does not ban peers") {
		t.Fatalf("policyNote must state no peer ban action: %q", note)
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}

	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
	}

	expectedFalse := []string{
		"bansPeer",
		"adjustsReputation",
		"connectsPeers",
		"disconnectsPeers",
		"replacesChain",
		"changesMempool",
		"changesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"startsMining",
	}

	for _, k := range expectedFalse {
		if safe[k] {
			t.Fatalf("safeBoundary.%s must be false", k)
		}
	}
}

func TestPeerReputationReviewV1DetectsHighRiskPeerPattern(t *testing.T) {
	out := AnalyzePeerReputation([]PeerReputationObservation{
		{
			PeerID:                "peer-bad",
			Score:                 10,
			Status:                "BAD",
			IsBanned:              true,
			InvalidMagicCount:     1,
			InvalidChainCount:     1,
			RejectedTxCount:       3,
			FloodOrOversizeCount:  1,
			ConnectionRejectCount: 1,
			UnknownMessageCount:   1,
			HandshakeFailureCount: 1,
		},
	})

	if out["overallRisk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["overallRisk"], out)
	}

	findings, ok := out["findings"].([]PeerReputationFinding)
	if !ok {
		t.Fatalf("findings type mismatch: %T", out["findings"])
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding")
	}

	top := findings[0]
	if top.PeerID != "peer-bad" {
		t.Fatalf("top peer mismatch: got=%s", top.PeerID)
	}
	if top.Risk != "HIGH" {
		t.Fatalf("top risk mismatch: got=%s score=%d reasons=%v", top.Risk, top.Score, top.Reasons)
	}
	if top.Score < 70 {
		t.Fatalf("top score should be >=70, got=%d", top.Score)
	}

	safe := out["safeBoundary"].(map[string]bool)
	if !safe["readOnly"] ||
		safe["bansPeer"] ||
		safe["adjustsReputation"] ||
		safe["connectsPeers"] ||
		safe["disconnectsPeers"] ||
		safe["replacesChain"] ||
		safe["changesMempool"] ||
		safe["changesChain"] ||
		safe["changesBalances"] ||
		safe["changesConsensus"] ||
		safe["changesTotalSupply"] ||
		safe["performsTransfer"] ||
		safe["startsMining"] {
		t.Fatalf("safeBoundary violated: %+v", safe)
	}
}

func TestPeerReputationReviewV1AdminReportSourceVisibility(t *testing.T) {
	data, err := os.ReadFile("admin_report.go")
	if err != nil {
		t.Fatalf("read admin_report.go: %v", err)
	}

	text := string(data)

	required := []string{
		"PeerReputation",
		"peerRisk",
		"peerReputation",
		"Review P2P peer reputation/ban-scoring warnings before public/testnet exposure.",
		"changesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"startsMining",
		"replacesChain",
	}

	for _, marker := range required {
		if !strings.Contains(text, marker) {
			t.Fatalf("admin_report.go missing marker %q", marker)
		}
	}
}
