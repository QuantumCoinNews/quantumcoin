package security

import "testing"

func TestIPAPIReputationV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := AnalyzeIPAPIReputation(nil)

	if out["version"] != IPAPIReputationVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], IPAPIReputationVersion)
	}
	if out["mode"] != "READ_ONLY_REPUTATION_ANALYSIS" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["overallRisk"] != "LOW" {
		t.Fatalf("empty observations should be LOW, got=%v", out["overallRisk"])
	}
	if out["observationsAnalyzed"] != 0 {
		t.Fatalf("empty observations count mismatch: got=%v", out["observationsAnalyzed"])
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}

	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
	}

	expectedFalse := []string{
		"bansClient",
		"blocksRequest",
		"changesRateLimits",
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

func TestIPAPIReputationV1DetectsHighRiskClientPattern(t *testing.T) {
	endpoints := []string{
		"/api/tx/send",
		"/api/mine",
		"/api/v1/withdraw/request",
		"/api/settlement/mark-pending",
		"/api/watch/claim",
		"/api/social/claim",
		"/api/admin/nft/monthly/candidates",
		"/api/admin/nft/monthly/select-winner",
		"/api/v1/tgwt/earn",
		"/api/v1/tgwt/verify",
		"/api/wallet/balance/test",
		"/api/mempool",
		"/api/security/action-burst/status",
		"/api/security/rate-limit/status",
		"/api/unknown/probe",
	}

	observations := make([]IPAPIReputationObservation, 0, 120)

	for i := 0; i < 120; i++ {
		observations = append(observations, IPAPIReputationObservation{
			ClientIP:    "203.0.113.45",
			Endpoint:    endpoints[i%len(endpoints)],
			Method:      "POST",
			StatusCode:  429,
			RateLimited: true,
			AuthFailed:  i%4 == 0,
			Blocked:     i%10 == 0,
		})
	}

	out := AnalyzeIPAPIReputation(observations)

	if out["version"] != IPAPIReputationVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], IPAPIReputationVersion)
	}
	if out["overallRisk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["overallRisk"], out)
	}
	if out["observationsAnalyzed"] != 120 {
		t.Fatalf("observations count mismatch: got=%v", out["observationsAnalyzed"])
	}

	findings, ok := out["findings"].([]IPAPIReputationFinding)
	if !ok {
		t.Fatalf("findings type mismatch: %T", out["findings"])
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding")
	}

	top := findings[0]
	if top.Subject != "203.0.113.45" {
		t.Fatalf("top finding subject mismatch: got=%s", top.Subject)
	}
	if top.Risk != "HIGH" {
		t.Fatalf("top finding risk mismatch: got=%s score=%d reasons=%v", top.Risk, top.Score, top.Reasons)
	}
	if top.Score < 70 {
		t.Fatalf("top finding score should be >=70, got=%d", top.Score)
	}
	if top.Counters["requestCount"] != 120 {
		t.Fatalf("requestCount mismatch: got=%d", top.Counters["requestCount"])
	}
	if top.Counters["endpointSpread"] < 10 {
		t.Fatalf("endpointSpread should be >=10, got=%d", top.Counters["endpointSpread"])
	}
	if top.Counters["rateLimited"] == 0 {
		t.Fatalf("rateLimited counter should be >0")
	}
	if top.Counters["authFailed"] == 0 {
		t.Fatalf("authFailed counter should be >0")
	}
	if top.Counters["blocked"] == 0 {
		t.Fatalf("blocked counter should be >0")
	}
	if top.Counters["sensitiveHits"] == 0 {
		t.Fatalf("sensitiveHits counter should be >0")
	}

	safe := out["safeBoundary"].(map[string]bool)
	if !safe["readOnly"] ||
		safe["bansClient"] ||
		safe["blocksRequest"] ||
		safe["changesRateLimits"] ||
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
