package nft

import (
	"testing"
	"time"
)

func TestMonthlyCandidateRejectsAddressOnlyV1(t *testing.T) {
	decision, err := EvaluateMonthlyRareNFTCandidate(
		"QC_TEST_ADDRESS",
		time.Date(2031, time.December, 10, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("address-only evaluation returned unexpected error: %v", err)
	}
	if decision.Eligible {
		t.Fatal("address-only candidate was eligible without a verified game session")
	}
}
