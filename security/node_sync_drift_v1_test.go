package security

import (
	"os"
	"strings"
	"testing"
)

func TestNodeSyncDriftDetectorV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := EvaluateNodeSyncDrift(NodeSyncDriftInput{
		LocalHeight:           100,
		NetworkHeight:         102,
		MaxPeerHeight:         102,
		MinPeerHeight:         99,
		PeerCount:             4,
		LatestBlockAgeSeconds: 60,
		DriftWarningBlocks:    3,
		DriftCriticalBlocks:   12,
	})

	if out["version"] != NodeSyncDriftDetectorVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], NodeSyncDriftDetectorVersion)
	}
	if out["mode"] != "READ_ONLY_NODE_SYNC_DRIFT_ANALYSIS" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["health"] != "OK" {
		t.Fatalf("health mismatch: got=%v", out["health"])
	}
	if out["syncStatus"] != "SYNC_OK" {
		t.Fatalf("syncStatus mismatch: got=%v", out["syncStatus"])
	}
	if out["risk"] != "LOW" {
		t.Fatalf("risk mismatch: got=%v", out["risk"])
	}
	if out["driftBlocks"] != 2 {
		t.Fatalf("driftBlocks mismatch: got=%v", out["driftBlocks"])
	}

	note, ok := out["policyNote"].(string)
	if !ok || note == "" {
		t.Fatalf("policyNote must exist")
	}
	if !strings.Contains(note, "does not connect peers") {
		t.Fatalf("policyNote must state no peer connection action: %q", note)
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}

	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
	}

	expectedFalse := []string{
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

func TestNodeSyncDriftDetectorV1DetectsCriticalBehindPattern(t *testing.T) {
	out := EvaluateNodeSyncDrift(NodeSyncDriftInput{
		LocalHeight:           10,
		NetworkHeight:         40,
		MaxPeerHeight:         42,
		MinPeerHeight:         39,
		PeerCount:             3,
		LatestBlockAgeSeconds: 21601,
		DriftWarningBlocks:    3,
		DriftCriticalBlocks:   12,
	})

	if out["version"] != NodeSyncDriftDetectorVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], NodeSyncDriftDetectorVersion)
	}
	if out["health"] != "DEGRADED" {
		t.Fatalf("expected DEGRADED health, got=%v out=%v", out["health"], out)
	}
	if out["syncStatus"] != "NODE_CRITICALLY_BEHIND" {
		t.Fatalf("expected NODE_CRITICALLY_BEHIND, got=%v out=%v", out["syncStatus"], out)
	}
	if out["risk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["risk"], out)
	}
	if out["networkHeight"] != 42 {
		t.Fatalf("networkHeight should use max peer height when higher, got=%v", out["networkHeight"])
	}
	if out["driftBlocks"] != 32 {
		t.Fatalf("driftBlocks mismatch: got=%v", out["driftBlocks"])
	}
	if out["absoluteDriftBlocks"] != 32 {
		t.Fatalf("absoluteDriftBlocks mismatch: got=%v", out["absoluteDriftBlocks"])
	}
	if out["score"].(int) < 70 {
		t.Fatalf("expected score >=70, got=%v", out["score"])
	}

	warnings, ok := out["warnings"].([]string)
	if !ok {
		t.Fatalf("warnings type mismatch: %T", out["warnings"])
	}

	requiredWarnings := []string{
		"local node height is critically behind network height",
		"latest local block is older than 6 hours",
	}

	for _, want := range requiredWarnings {
		found := false
		for _, got := range warnings {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing warning %q in %v", want, warnings)
		}
	}

	safe := out["safeBoundary"].(map[string]bool)
	if !safe["readOnly"] ||
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

func TestNodeSyncDriftDetectorV1DetectsIsolatedNode(t *testing.T) {
	out := EvaluateNodeSyncDrift(NodeSyncDriftInput{
		LocalHeight:           25,
		NetworkHeight:         25,
		PeerCount:             0,
		LatestBlockAgeSeconds: 120,
	})

	if out["health"] != "ISOLATED" {
		t.Fatalf("expected ISOLATED health, got=%v out=%v", out["health"], out)
	}
	if out["syncStatus"] != "NO_PEERS_LOCAL_ONLY" {
		t.Fatalf("expected NO_PEERS_LOCAL_ONLY, got=%v out=%v", out["syncStatus"], out)
	}
	if out["risk"] != "WATCH" {
		t.Fatalf("expected WATCH risk, got=%v out=%v", out["risk"], out)
	}
}

func TestNodeSyncDriftDetectorV1AdminReportSourceVisibility(t *testing.T) {
	data, err := os.ReadFile("admin_report.go")
	if err != nil {
		t.Fatalf("read admin_report.go: %v", err)
	}

	text := string(data)

	required := []string{
		"NodeHealth",
		"nodeHealth",
		"nodeSync",
		"NO_PEERS_LOCAL_ONLY",
		"Connect at least one trusted peer before public/testnet use.",
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
