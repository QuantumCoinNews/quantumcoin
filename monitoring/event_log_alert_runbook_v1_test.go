package monitoring

import "testing"

func TestMonitoringEventValidationAndClassification(t *testing.T) {
	e := NewEvent("evt1", "node-api", "api_down", SeverityCritical, "node API unreachable")
	if err := ValidateEvent(e); err != nil {
		t.Fatalf("event should validate: %v", err)
	}

	alert := ClassifyAlert(e)
	if !alert.Open {
		t.Fatalf("critical event must open alert")
	}
	if !alert.Escalate {
		t.Fatalf("critical event must escalate")
	}
	if alert.SafeBoundary == "" {
		t.Fatalf("safe boundary required")
	}
}

func TestMonitoringRejectsUnnormalizedSeverity(t *testing.T) {
	e := Event{
		Source:   "admin",
		Category: "auth_failure",
		Severity: Severity("warn"),
		Message:  "bad token",
	}
	if err := ValidateEvent(e); err == nil {
		t.Fatalf("unnormalized severity must fail")
	}
}

func TestRunbookContainsCriticalCategories(t *testing.T) {
	entries := DefaultRunbookEntries()
	required := map[string]bool{
		"api_down":                   false,
		"auth_failure":               false,
		"chain_mismatch":             false,
		"payout_guard":               false,
		"nft_mint_guard":             false,
		"watch_verification_failure": false,
	}

	for _, e := range entries {
		if _, ok := required[e.Category]; ok {
			required[e.Category] = true
		}
		if len(e.Forbidden) == 0 {
			t.Fatalf("runbook entry must include forbidden actions: %+v", e)
		}
	}

	for k, ok := range required {
		if !ok {
			t.Fatalf("missing runbook category: %s", k)
		}
	}
}

func TestSafeBoundary(t *testing.T) {
	b := SafeBoundaryV1()
	for _, k := range []string{
		"doesNotMint",
		"doesNotSendFunds",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeSupply",
		"doesNotStartMining",
		"doesNotStopMining",
	} {
		if !b[k] {
			t.Fatalf("safe boundary missing: %s", k)
		}
	}
}
