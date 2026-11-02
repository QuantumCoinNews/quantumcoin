package nft

import (
	"errors"
	"testing"
)

func TestMonthlyWinnerRejectsDuplicateEligibleUserV1(t *testing.T) {
	a := MonthlyNFTCandidateV1{UserID: "user-a", WalletAddress: "QC_A", Period: "2032-01", Eligible: true, AIApproved: true, AdminApproved: true}
	for _, wallet := range []string{"QC_A", "QC_OTHER"} {
		b := a
		b.WalletAddress = wallet
		if _, err := SelectMonthlyNFTWinnerPlaceholderV1(a.Period, "seed", []MonthlyNFTCandidateV1{a, b}); !errors.Is(err, ErrMonthlyNFTInvalidCandidate) {
			t.Fatalf("duplicate user accepted: %v", err)
		}
	}
}

func TestMonthlyWinnerRejectsDuplicateEligibleWalletV1(t *testing.T) {
	a := MonthlyNFTCandidateV1{UserID: "user-a", WalletAddress: "QC_A", Period: "2032-01", Eligible: true, AIApproved: true, AdminApproved: true}
	b := a
	b.UserID = "user-b"
	if _, err := SelectMonthlyNFTWinnerPlaceholderV1(a.Period, "seed", []MonthlyNFTCandidateV1{a, b}); !errors.Is(err, ErrMonthlyNFTInvalidCandidate) {
		t.Fatalf("duplicate wallet accepted: %v", err)
	}
}

func TestMonthlyWinnerRejectsWhitespaceIdentityAliasesV1(t *testing.T) {
	a := MonthlyNFTCandidateV1{UserID: "user-a", WalletAddress: "QC_A", Period: "2032-01", Eligible: true, AIApproved: true, AdminApproved: true}
	b := a
	b.UserID, b.WalletAddress = " user-a ", " QC_A "
	if _, err := SelectMonthlyNFTWinnerPlaceholderV1(a.Period, "seed", []MonthlyNFTCandidateV1{a, b}); !errors.Is(err, ErrMonthlyNFTInvalidCandidate) {
		t.Fatalf("whitespace duplicate accepted: %v", err)
	}
}

func TestMonthlyWinnerIneligibleDuplicateHasNoWeightV1(t *testing.T) {
	a := MonthlyNFTCandidateV1{UserID: "user-a", WalletAddress: "QC_A", Period: "2032-01", Eligible: true, AIApproved: true, AdminApproved: true}
	baseline, err := SelectMonthlyNFTWinnerPlaceholderV1(a.Period, "seed", []MonthlyNFTCandidateV1{a})
	if err != nil {
		t.Fatal(err)
	}
	b := a
	b.AIApproved = false
	actual, err := SelectMonthlyNFTWinnerPlaceholderV1(a.Period, "seed", []MonthlyNFTCandidateV1{a, b})
	if err != nil || actual != baseline {
		t.Fatalf("ineligible row changed decision: %v", err)
	}
}
