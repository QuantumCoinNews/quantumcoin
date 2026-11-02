package security

import (
	"os"
	"strings"
	"testing"
)

func TestMinerSafetyGuardReviewV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := AnalyzeMinerSafety([]MinerSafetyObservation{
		{
			Subject:             "release-api-mempool-miner",
			Mode:                "visible CMD -> API /api/mine loop",
			LoopActive:          true,
			StopFlagPresent:     true,
			DelayMs:             5000,
			ManualStopObserved:  true,
			APIStartAttempted:   true,
			VisibleCMDMode:      true,
			StableReleaseScript: true,
		},
	})

	if out["version"] != MinerSafetyGuardReviewVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], MinerSafetyGuardReviewVersion)
	}
	if out["mode"] != "READ_ONLY_MINER_SAFETY_REVIEW" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["overallRisk"] != "LOW" {
		t.Fatalf("risk mismatch: got=%v out=%v", out["overallRisk"], out)
	}

	note, ok := out["policyNote"].(string)
	if !ok || note == "" {
		t.Fatalf("policyNote must exist")
	}
	if !strings.Contains(note, "does not start mining") {
		t.Fatalf("policyNote must state no mining start action: %q", note)
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}

	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
	}

	expectedFalse := []string{
		"startsMining",
		"stopsMining",
		"killsProcess",
		"changesMiningReward",
		"changesMempool",
		"changesChain",
		"replacesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"modifiesReleaseScript",
	}

	for _, k := range expectedFalse {
		if safe[k] {
			t.Fatalf("safeBoundary.%s must be false", k)
		}
	}
}

func TestMinerSafetyGuardReviewV1DetectsHighRiskLoopPattern(t *testing.T) {
	out := AnalyzeMinerSafety([]MinerSafetyObservation{
		{
			Subject:               "miner-loop-risk",
			Mode:                  "api-mempool-loop",
			LoopActive:            true,
			StopFlagPresent:       false,
			DelayMs:               0,
			DuplicateMinerCount:   2,
			APIMissingCount:       2,
			MineErrorCount:        3,
			ConsecutiveErrorCount: 10,
			MissingAddressCount:   1,
			UnexpectedExitCount:   1,
			PanicOrCrashCount:     1,
			StableReleaseScript:   false,
		},
	})

	if out["overallRisk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["overallRisk"], out)
	}

	findings, ok := out["findings"].([]MinerSafetyFinding)
	if !ok {
		t.Fatalf("findings type mismatch: %T", out["findings"])
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding")
	}

	top := findings[0]
	if top.Subject != "miner-loop-risk" {
		t.Fatalf("top subject mismatch: got=%s", top.Subject)
	}
	if top.Risk != "HIGH" {
		t.Fatalf("top risk mismatch: got=%s score=%d reasons=%v", top.Risk, top.Score, top.Reasons)
	}
	if top.Score < 70 {
		t.Fatalf("top score should be >=70, got=%d", top.Score)
	}
	if top.Counters["duplicateMinerCount"] != 2 {
		t.Fatalf("duplicateMinerCount counter mismatch: %+v", top.Counters)
	}
	if top.Counters["consecutiveErrorCount"] != 10 {
		t.Fatalf("consecutiveErrorCount counter mismatch: %+v", top.Counters)
	}
	if top.Counters["panicOrCrashCount"] != 1 {
		t.Fatalf("panicOrCrashCount counter mismatch: %+v", top.Counters)
	}

	safe := out["safeBoundary"].(map[string]bool)
	if !safe["readOnly"] ||
		safe["startsMining"] ||
		safe["stopsMining"] ||
		safe["killsProcess"] ||
		safe["changesMiningReward"] ||
		safe["changesMempool"] ||
		safe["changesChain"] ||
		safe["replacesChain"] ||
		safe["changesBalances"] ||
		safe["changesConsensus"] ||
		safe["changesTotalSupply"] ||
		safe["performsTransfer"] ||
		safe["modifiesReleaseScript"] {
		t.Fatalf("safeBoundary violated: %+v", safe)
	}
}

func TestMinerSafetyGuardReviewV1AdminReportSourceVisibility(t *testing.T) {
	data, err := os.ReadFile("admin_report.go")
	if err != nil {
		t.Fatalf("read admin_report.go: %v", err)
	}

	text := string(data)

	required := []string{
		"MinerSafety",
		"minerSafetyRisk",
		"minerSafety",
		"Review miner loop/process safety warnings before public/testnet exposure.",
		"changesMempool",
		"changesChain",
		"replacesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"startsMining",
	}

	for _, marker := range required {
		if !strings.Contains(text, marker) {
			t.Fatalf("admin_report.go missing marker %q", marker)
		}
	}
}
