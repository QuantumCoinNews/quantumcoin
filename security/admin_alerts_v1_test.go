package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminSecurityAlertsV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := BuildAdminSecurityAlerts(AdminSecurityAlertInput{
		AdminReport: map[string]any{
			"overallRisk":   "LOW",
			"overallStatus": "OK",
			"securityScore": 95,
		},
		EventLog: map[string]any{
			"overallRisk": "LOW",
			"totalEvents": 0,
		},
	})

	if out["version"] != AdminSecurityAlertsVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], AdminSecurityAlertsVersion)
	}
	if out["mode"] != "READ_ONLY_ADMIN_SECURITY_ALERTS" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["overallAlertLevel"] != "INFO" {
		t.Fatalf("overallAlertLevel mismatch: got=%v out=%v", out["overallAlertLevel"], out)
	}

	alerts, ok := out["alerts"].([]map[string]any)
	if !ok || len(alerts) != 1 {
		t.Fatalf("expected one info alert, got=%T %v", out["alerts"], out["alerts"])
	}
	if alerts[0]["type"] != "NO_IMMEDIATE_ADMIN_ALERT" {
		t.Fatalf("unexpected alert: %+v", alerts[0])
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}
	if !safe["readOnly"] || !safe["advisoryOnly"] {
		t.Fatalf("safeBoundary must be readOnly/advisoryOnly: %+v", safe)
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
		"stopsMining",
		"killsProcess",
		"modifiesReleaseExecutable",
	}

	for _, k := range expectedFalse {
		if safe[k] {
			t.Fatalf("safeBoundary.%s must be false", k)
		}
	}
}

func TestAdminSecurityAlertsV1HighRiskAndCriticalEvent(t *testing.T) {
	out := BuildAdminSecurityAlerts(AdminSecurityAlertInput{
		AdminReport: map[string]any{
			"overallRisk":   "HIGH",
			"overallStatus": "CRITICAL_REVIEW_REQUIRED",
			"securityScore": 50,
		},
		EventLog: map[string]any{
			"overallRisk": "CRITICAL",
			"totalEvents": 2,
		},
	})

	if out["overallAlertLevel"] != "CRITICAL" {
		t.Fatalf("expected CRITICAL alert level, got=%v out=%v", out["overallAlertLevel"], out)
	}

	alerts, ok := out["alerts"].([]map[string]any)
	if !ok {
		t.Fatalf("alerts type mismatch: %T", out["alerts"])
	}

	seenAdmin := false
	seenEvent := false
	seenScore := false

	for _, a := range alerts {
		switch a["type"] {
		case "ADMIN_REPORT_HIGH_RISK":
			seenAdmin = true
		case "SECURITY_EVENT_CRITICAL":
			seenEvent = true
		case "LOW_SECURITY_SCORE":
			seenScore = true
		}
	}

	if !seenAdmin || !seenEvent || !seenScore {
		t.Fatalf("expected admin/event/score alerts, got=%+v", alerts)
	}

	safe := out["safeBoundary"].(map[string]bool)
	if !safe["readOnly"] ||
		!safe["advisoryOnly"] ||
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
		safe["startsMining"] ||
		safe["stopsMining"] ||
		safe["killsProcess"] ||
		safe["modifiesReleaseExecutable"] {
		t.Fatalf("safeBoundary violated: %+v", safe)
	}
}

func TestSecurityEventLogV1AppendReadAndSummary(t *testing.T) {
	dir := t.TempDir()

	ev := NewEvent(
		"ADMIN_ALERT_TEST",
		SeverityCritical,
		"admin_alerts_test",
		"wallet",
		"critical test event",
		map[string]any{"case": "admin_alerts_v1"},
	)

	if err := AppendEvent(dir, ev); err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}

	payload := ReadEvents(dir, 10)
	if payload.Version != SecurityEventLogVersion {
		t.Fatalf("event log version mismatch: got=%s", payload.Version)
	}
	if len(payload.Events) != 1 {
		t.Fatalf("expected 1 event, got=%d payload=%+v", len(payload.Events), payload)
	}

	summary := SummarizeEvents(payload.Events)
	if summary["version"] != SecurityEventLogVersion {
		t.Fatalf("summary version mismatch: got=%v", summary["version"])
	}
	if summary["overallRisk"] != "CRITICAL" {
		t.Fatalf("summary overallRisk mismatch: got=%v summary=%v", summary["overallRisk"], summary)
	}
	if summary["readOnly"] != true || summary["advisoryOnly"] != true {
		t.Fatalf("summary must be readOnly/advisoryOnly: %v", summary)
	}

	path := filepath.Join(dir, "security_events.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("security event log file should exist at %s: %v", path, err)
	}
}

func TestAdminSecurityAlertsV1SourceMarkers(t *testing.T) {
	files := []string{
		"admin_alerts.go",
		"event_log.go",
		"event_log_summary.go",
	}

	required := []string{
		"ADMIN_SECURITY_ALERTS_V1",
		"READ_ONLY_ADMIN_SECURITY_ALERTS",
		"SECURITY_EVENT_LOG_V1",
		"SECURITY_EVENT_SUMMARY_V1",
		"NewEvent",
		"AppendEvent",
		"ReadEvents",
		"SummarizeEvents",
		"severity",
		"overallAlertLevel",
		"policyNote",
		"safeBoundary",
		"readOnly",
		"advisoryOnly",
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
		"stopsMining",
		"killsProcess",
		"modifiesReleaseExecutable",
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
			t.Fatalf("missing source marker %q", marker)
		}
	}
}
