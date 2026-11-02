package nft

import (
	"testing"
	"time"
)

func TestMonthlyNFTPeriodsFromActivationV1(t *testing.T) {
	activatedAt := time.Date(2031, time.November, 17, 12, 0, 0, 0, time.UTC)

	periods, err := MonthlyNFTPeriodsFromActivationV1(activatedAt)
	if err != nil {
		t.Fatalf("build periods: %v", err)
	}

	if len(periods) != 12 {
		t.Fatalf("period count = %d, want 12", len(periods))
	}
	if periods[0] != "2031-12" {
		t.Fatalf("first full month = %q, want 2031-12", periods[0])
	}
	if periods[11] != "2032-11" {
		t.Fatalf("twelfth month = %q, want 2032-11", periods[11])
	}

	seen := make(map[string]bool, 12)
	for _, period := range periods {
		if seen[period] {
			t.Fatalf("duplicate reward period %q", period)
		}
		seen[period] = true
	}
}

func TestMonthlyNFTPeriodsRejectMissingActivationV1(t *testing.T) {
	if _, err := MonthlyNFTPeriodsFromActivationV1(time.Time{}); err == nil {
		t.Fatal("missing mainnet NFT activation was accepted")
	}
}
