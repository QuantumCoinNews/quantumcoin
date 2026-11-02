package security

import (
	"strings"
	"testing"
)

func TestFinalityCheckpointWarningV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := EvaluateFinalityCheckpointWarning(FinalityCheckpointInput{
		ChainHeight:           100,
		BlockCount:            101,
		LatestBlockHash:       "abc123",
		LatestBlockAgeSeconds: 60,
		PeerCount:             3,
		FinalityDepth:         6,
		ReorgWarningDepth:     3,
	})

	if out["version"] != FinalityCheckpointWarningVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], FinalityCheckpointWarningVersion)
	}
	if out["mode"] != "READ_ONLY_WARNING" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["status"] != "FINALITY_OK" {
		t.Fatalf("status mismatch: got=%v", out["status"])
	}
	if out["risk"] != "LOW" {
		t.Fatalf("risk mismatch: got=%v", out["risk"])
	}
	if out["confirmedDepth"] != 94 {
		t.Fatalf("confirmedDepth mismatch: got=%v", out["confirmedDepth"])
	}

	note, ok := out["policyNote"].(string)
	if !ok || note == "" {
		t.Fatalf("policyNote must exist")
	}
	if !strings.Contains(note, "does not replace chain") {
		t.Fatalf("policyNote must state no chain replacement: %q", note)
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}

	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
	}

	expectedFalse := []string{
		"replacesChain",
		"rejectsBlocks",
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

func TestFinalityCheckpointWarningV1DetectsHighRiskFinalityPattern(t *testing.T) {
	out := EvaluateFinalityCheckpointWarning(FinalityCheckpointInput{
		ChainHeight:           2,
		BlockCount:            0,
		LatestBlockHash:       "",
		LatestBlockAgeSeconds: 21601,
		PeerCount:             0,
		FinalityDepth:         6,
		ReorgWarningDepth:     3,
	})

	if out["version"] != FinalityCheckpointWarningVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], FinalityCheckpointWarningVersion)
	}
	if out["status"] != "STALE_CHAIN_FINALITY_WARNING" {
		t.Fatalf("expected stale chain status, got=%v out=%v", out["status"], out)
	}
	if out["risk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["risk"], out)
	}
	if out["score"].(int) < 70 {
		t.Fatalf("expected score >=70, got=%v", out["score"])
	}

	warnings, ok := out["warnings"].([]string)
	if !ok {
		t.Fatalf("warnings type mismatch: %T", out["warnings"])
	}

	requiredWarnings := []string{
		"no chain data available",
		"chain height is below configured finality depth",
		"node has no connected peers; finality view is local only",
		"latest block is older than 6 hours",
		"latest block hash is empty",
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
		safe["replacesChain"] ||
		safe["rejectsBlocks"] ||
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

func TestFinalityCheckpointWarningV1DefaultsAndConfirmedDepth(t *testing.T) {
	out := EvaluateFinalityCheckpointWarning(FinalityCheckpointInput{
		ChainHeight:           5,
		BlockCount:            6,
		LatestBlockHash:       "headhash",
		LatestBlockAgeSeconds: 120,
		PeerCount:             1,
	})

	if out["finalityDepth"] != 6 {
		t.Fatalf("default finalityDepth mismatch: got=%v", out["finalityDepth"])
	}
	if out["reorgWarningDepth"] != 3 {
		t.Fatalf("default reorgWarningDepth mismatch: got=%v", out["reorgWarningDepth"])
	}
	if out["confirmedDepth"] != 0 {
		t.Fatalf("confirmedDepth should not be negative, got=%v", out["confirmedDepth"])
	}
	if out["risk"] != "WATCH" {
		t.Fatalf("height below finality depth should be WATCH, got=%v out=%v", out["risk"], out)
	}
}
