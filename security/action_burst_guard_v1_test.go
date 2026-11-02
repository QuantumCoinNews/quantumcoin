package security

import (
	"os"
	"strings"
	"testing"
)

func TestActionBurstGuardReviewV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := AnalyzeActionBurst([]ActionBurstObservation{
		{
			Subject:       "client-good",
			ClientIP:      "198.51.100.10",
			PlayerID:      "player-good",
			Endpoint:      "/api/security/action-burst/test-ping",
			WindowSeconds: 60,
			RequestCount:  2,
		},
	})

	if out["version"] != ActionBurstGuardReviewVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], ActionBurstGuardReviewVersion)
	}
	if out["mode"] != "READ_ONLY_ACTION_BURST_RATE_LIMIT_REVIEW" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["overallRisk"] != "LOW" {
		t.Fatalf("risk mismatch: got=%v out=%v", out["overallRisk"], out)
	}

	note, ok := out["policyNote"].(string)
	if !ok || note == "" {
		t.Fatalf("policyNote must exist")
	}
	if !strings.Contains(note, "does not block requests") {
		t.Fatalf("policyNote must state no request blocking action: %q", note)
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}

	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
	}

	expectedFalse := []string{
		"blocksRequest",
		"bansClient",
		"changesRateLimits",
		"changesMempool",
		"changesChain",
		"replacesChain",
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

func TestActionBurstGuardReviewV1DetectsHighRiskBurstPattern(t *testing.T) {
	out := AnalyzeActionBurst([]ActionBurstObservation{
		{
			Subject:            "player-burst",
			ClientIP:           "198.51.100.11",
			PlayerID:           "player-burst",
			Endpoint:           "/api/security/action-burst/test-ping",
			WindowSeconds:      10,
			RequestCount:       200,
			RateLimitedCount:   3,
			Status429Count:     3,
			BlockedCount:       2,
			CooldownHitCount:   2,
			DailyLimitHitCount: 1,
			SensitiveEndpoint:  true,
			P2PFloodCount:      1,
		},
	})

	if out["overallRisk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["overallRisk"], out)
	}

	findings, ok := out["findings"].([]ActionBurstFinding)
	if !ok {
		t.Fatalf("findings type mismatch: %T", out["findings"])
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding")
	}

	top := findings[0]
	if top.Subject != "player-burst" {
		t.Fatalf("top subject mismatch: got=%s", top.Subject)
	}
	if top.Risk != "HIGH" {
		t.Fatalf("top risk mismatch: got=%s score=%d reasons=%v", top.Risk, top.Score, top.Reasons)
	}
	if top.Score < 70 {
		t.Fatalf("top score should be >=70, got=%d", top.Score)
	}
	if top.Counters["requestCount"] != 200 {
		t.Fatalf("requestCount counter mismatch: %+v", top.Counters)
	}
	if top.Counters["rateLimitedCount"] != 3 {
		t.Fatalf("rateLimitedCount counter mismatch: %+v", top.Counters)
	}
	if top.Counters["status429Count"] != 3 {
		t.Fatalf("status429Count counter mismatch: %+v", top.Counters)
	}
	if top.Counters["blockedCount"] != 2 {
		t.Fatalf("blockedCount counter mismatch: %+v", top.Counters)
	}

	safe := out["safeBoundary"].(map[string]bool)
	if !safe["readOnly"] ||
		safe["blocksRequest"] ||
		safe["bansClient"] ||
		safe["changesRateLimits"] ||
		safe["changesMempool"] ||
		safe["changesChain"] ||
		safe["replacesChain"] ||
		safe["changesBalances"] ||
		safe["changesConsensus"] ||
		safe["changesTotalSupply"] ||
		safe["performsTransfer"] ||
		safe["startsMining"] {
		t.Fatalf("safeBoundary violated: %+v", safe)
	}
}

func TestActionBurstGuardReviewV1AdminReportSourceVisibility(t *testing.T) {
	data, err := os.ReadFile("admin_report.go")
	if err != nil {
		t.Fatalf("read admin_report.go: %v", err)
	}

	text := string(data)

	required := []string{
		"ActionBurstGuard",
		"actionBurstRisk",
		"actionBurstGuard",
		"Review API/TG/P2P rate-limit and action-burst warnings before public/testnet exposure.",
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
