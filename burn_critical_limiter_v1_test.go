package main

import (
	"os"
	"strings"
	"testing"
)

// QMA-0062: explicit burn is a critical HTTP action.
// This is API anti-abuse only; it is not a consensus identity rule.
func TestCanonicalBurnRouteUsesCriticalLimiterV1(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	source := string(data)

	required := []string{
		`"/api/tx/burn",`,
		`"tx_burn",`,
		`protected(handleBurn),`,
	}

	for _, item := range required {
		if !strings.Contains(source, item) {
			t.Fatalf(
				"canonical burn critical limiter contract missing %q",
				item,
			)
		}
	}
}
