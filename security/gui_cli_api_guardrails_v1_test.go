package security

import (
	"os"
	"strings"
	"testing"
)

func TestGUICLIAPIGuardrailsReviewV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := AnalyzeGUICLIAPIGuardrails([]GUICLIAPIGuardrailObservation{
		{
			Subject:            "wallet-gui-canonical-api",
			Surface:            "GUI",
			APIBase:            "http://127.0.0.1:8082",
			Endpoint:           "/api/tx/send",
			UsesLocalhost:      true,
			UsesAPIBalance:     true,
			UsesTxSendEndpoint: true,
		},
	})

	if out["version"] != GUICLIAPIGuardrailsReviewVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], GUICLIAPIGuardrailsReviewVersion)
	}
	if out["mode"] != "READ_ONLY_GUI_CLI_API_GUARDRAILS_REVIEW" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["overallRisk"] != "LOW" {
		t.Fatalf("risk mismatch: got=%v out=%v", out["overallRisk"], out)
	}

	note, ok := out["policyNote"].(string)
	if !ok || note == "" {
		t.Fatalf("policyNote must exist")
	}
	if !strings.Contains(note, "does not change GUI behavior") {
		t.Fatalf("policyNote must state no GUI mutation action: %q", note)
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}
	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
	}

	expectedFalse := []string{
		"changesGUI",
		"changesCLI",
		"changesAPI",
		"changesEndpoint",
		"blocksRequest",
		"startsMining",
		"stopsMining",
		"killsProcess",
		"performsTransfer",
		"changesBalances",
		"changesMempool",
		"changesChain",
		"replacesChain",
		"changesConsensus",
		"changesTotalSupply",
		"modifiesReleaseExecutable",
	}

	for _, k := range expectedFalse {
		if safe[k] {
			t.Fatalf("safeBoundary.%s must be false", k)
		}
	}
}

func TestGUICLIAPIGuardrailsReviewV1DetectsHighRiskSurfacePattern(t *testing.T) {
	out := AnalyzeGUICLIAPIGuardrails([]GUICLIAPIGuardrailObservation{
		{
			Subject:                 "gui-cache-fallback-risk",
			Surface:                 "GUI",
			APIBase:                 "http://127.0.0.1:3000",
			Endpoint:                "/api/send",
			UsesLocalhost:           true,
			UsesUnexpectedPort:      true,
			UsesBalanceCache:        true,
			UsesMinedBalanceLocal:   true,
			UsesBonusStoreLocal:     true,
			UsesSendFallback:        true,
			StartsProcess:           true,
			KillsProcess:            true,
			MissingValidationCount:  2,
			EndpointFallbackCount:   2,
			CriticalEndpoint:        true,
			TransferEndpoint:        true,
			TimeoutOrAPIErrorCount:  2,
			StaleCacheFallbackCount: 2,
		},
	})

	if out["overallRisk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["overallRisk"], out)
	}

	findings, ok := out["findings"].([]GUICLIAPIGuardrailFinding)
	if !ok {
		t.Fatalf("findings type mismatch: %T", out["findings"])
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding")
	}
	if findings[0].Risk != "HIGH" {
		t.Fatalf("top risk mismatch: got=%s score=%d", findings[0].Risk, findings[0].Score)
	}
	if findings[0].Score < 70 {
		t.Fatalf("top score should be >=70, got=%d", findings[0].Score)
	}
}

func TestGUICLIAPIGuardrailsAdminVisibilityV1(t *testing.T) {
	review := map[string]any{
		"overallRisk": "HIGH",
	}

	out := BuildGUICLIAPIGuardrailsAdminVisibility(review)

	if out["version"] != GUICLIAPIGuardrailsAdminVisibilityVersion {
		t.Fatalf("version mismatch: got=%v", out["version"])
	}
	if out["guiCliApiRisk"] != "HIGH" {
		t.Fatalf("guiCliApiRisk mismatch: got=%v", out["guiCliApiRisk"])
	}
	if out["guiCliApiGuardrails"] == nil {
		t.Fatalf("guiCliApiGuardrails should be visible")
	}
	if out["recommendedAction"] != "Review GUI/CLI/API guardrail warnings before release/testnet exposure." {
		t.Fatalf("recommendedAction mismatch: got=%v", out["recommendedAction"])
	}

	falseKeys := []string{
		"changesMempool",
		"changesChain",
		"replacesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"startsMining",
		"changesGUI",
		"changesCLI",
		"changesAPI",
		"blocksRequest",
	}

	for _, k := range falseKeys {
		if out[k] != false {
			t.Fatalf("%s must be false; out=%v", k, out)
		}
	}
}

func TestGUICLIAPIGuardrailsReviewV1SourceVisibility(t *testing.T) {
	files := []string{
		"gui_cli_api_guardrails.go",
		"gui_cli_api_admin_visibility.go",
	}

	required := []string{
		"GUI_CLI_API_GUARDRAILS_REVIEW_V1",
		"AnalyzeGUICLIAPIGuardrails",
		"READ_ONLY_GUI_CLI_API_GUARDRAILS_REVIEW",
		"GUI_CLI_API_GUARDRAILS_ADMIN_VISIBILITY_V1",
		"BuildGUICLIAPIGuardrailsAdminVisibility",
		"guiCliApiRisk",
		"guiCliApiGuardrails",
		"Review GUI/CLI/API guardrail warnings before release/testnet exposure.",
		"changesMempool",
		"changesChain",
		"replacesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"startsMining",
		"changesGUI",
		"changesCLI",
		"changesAPI",
		"blocksRequest",
	}

	all := ""
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		all += string(data) + "\n"
	}

	for _, marker := range required {
		if !strings.Contains(all, marker) {
			t.Fatalf("source visibility missing marker %q", marker)
		}
	}
}
