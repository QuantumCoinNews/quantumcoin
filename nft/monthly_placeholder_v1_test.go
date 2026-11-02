package nft

import "testing"

func TestBuildMonthlyRareNFTPlaceholderPlanV1(t *testing.T) {
	plan := BuildMonthlyRareNFTPlaceholderPlanV1(2026)

	if plan.Version != MonthlyNFTPlaceholderVersionV1 {
		t.Fatalf("unexpected version: %s", plan.Version)
	}
	if plan.TotalSlots != 12 || len(plan.Slots) != 12 {
		t.Fatalf("expected 12 slots, got total=%d len=%d", plan.TotalSlots, len(plan.Slots))
	}
	if err := ValidateMonthlyNFTPlanV1(plan); err != nil {
		t.Fatalf("plan should validate: %v", err)
	}

	for _, slot := range plan.Slots {
		if slot.Rarity != "rare" {
			t.Fatalf("slot must be rare: %+v", slot)
		}
		if slot.MintEnabled || slot.Minted {
			t.Fatalf("placeholder must not mint: %+v", slot)
		}
	}
}

func TestMonthlyNFTPlanRejectsDuplicates(t *testing.T) {
	plan := BuildMonthlyRareNFTPlaceholderPlanV1(2026)
	plan.Slots[1].Name = plan.Slots[0].Name

	if err := ValidateMonthlyNFTPlanV1(plan); err == nil {
		t.Fatalf("expected duplicate name to fail")
	}
}

func TestMonthlyNFTMainnetGateRequiresApproval(t *testing.T) {
	plan := BuildMonthlyRareNFTPlaceholderPlanV1(2026)
	plan.Mode = "mainnet"

	if err := ValidateMonthlyNFTMainnetMintGateV1(plan); err != ErrMonthlyNFTInvalidMainnet {
		t.Fatalf("expected mainnet approval error, got %v", err)
	}

	plan.MainnetMintApproval = MonthlyNFTMainnetMintApprovalV1
	if err := ValidateMonthlyNFTMainnetMintGateV1(plan); err != ErrMonthlyNFTMintNotAllowed {
		t.Fatalf("public approval token must not authorize a placeholder plan: %v", err)
	}
}

func TestSelectMonthlyNFTWinnerPlaceholderV1(t *testing.T) {
	candidates := []MonthlyNFTCandidateV1{
		{UserID: "user-b", WalletAddress: "QC_B", Period: "2026-06", Eligible: true, AIApproved: true, AdminApproved: true},
		{UserID: "user-a", WalletAddress: "QC_A", Period: "2026-06", Eligible: true, AIApproved: true, AdminApproved: true},
		{UserID: "user-x", WalletAddress: "QC_X", Period: "2026-07", Eligible: true, AIApproved: true, AdminApproved: true},
		{UserID: "user-c", WalletAddress: "QC_C", Period: "2026-06", Eligible: true, AIApproved: false, AdminApproved: true},
	}

	decision, err := SelectMonthlyNFTWinnerPlaceholderV1("2026-06", "seed-1", candidates)
	if err != nil {
		t.Fatalf("winner selection failed: %v", err)
	}
	if decision.Version != MonthlyNFTPlaceholderVersionV1 {
		t.Fatalf("bad version: %s", decision.Version)
	}
	if decision.Period != "2026-06" {
		t.Fatalf("bad period: %s", decision.Period)
	}
	if decision.WinnerUserID == "" || decision.WalletAddress == "" || decision.NFTID == "" || decision.SelectionHash == "" {
		t.Fatalf("winner decision missing fields: %+v", decision)
	}
	if decision.MintAllowed || decision.Minted {
		t.Fatalf("placeholder selection must not mint: %+v", decision)
	}

	again, err := SelectMonthlyNFTWinnerPlaceholderV1("2026-06", "seed-1", candidates)
	if err != nil {
		t.Fatalf("repeat selection failed: %v", err)
	}
	if again.WinnerUserID != decision.WinnerUserID || again.SelectionHash != decision.SelectionHash {
		t.Fatalf("selection must be deterministic for same seed")
	}
}

func TestSelectMonthlyNFTWinnerNoCandidates(t *testing.T) {
	_, err := SelectMonthlyNFTWinnerPlaceholderV1("2026-06", "seed", nil)
	if err != ErrMonthlyNFTNoCandidates {
		t.Fatalf("expected no candidates error, got %v", err)
	}
}
