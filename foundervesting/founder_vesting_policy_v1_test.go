package foundervesting

import "testing"

func TestDefaultFounderVestingPolicyValid(t *testing.T) {
	policy := DefaultPolicyV1()

	if err := ValidatePolicyV1(policy); err != nil {
		t.Fatalf("default founder vesting policy should validate: %v", err)
	}
}

func TestFounderAllocationSplit(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.FounderTotalQC != 1275000 {
		t.Fatalf("founder total must be 1,275,000 QC")
	}
	if policy.FounderLiquidQC != 255000 {
		t.Fatalf("founder liquid must be 255,000 QC")
	}
	if policy.FounderLockedQC != 1020000 {
		t.Fatalf("founder locked must be 1,020,000 QC")
	}
	if policy.FounderLiquidQC+policy.FounderLockedQC != policy.FounderTotalQC {
		t.Fatalf("founder liquid + locked must equal total")
	}
}

func TestFounderVestingSchedule(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.CliffMonths != 12 {
		t.Fatalf("cliff must be 12 months")
	}
	if policy.VestingMonths != 48 {
		t.Fatalf("vesting must be 48 months")
	}
	if policy.MonthlyUnlockQC != 21250 {
		t.Fatalf("monthly unlock must be 21,250 QC")
	}
	if policy.MonthlyUnlockQC*int64(policy.VestingMonths) != policy.FounderLockedQC {
		t.Fatalf("48 monthly unlocks must equal locked allocation")
	}
}

func TestFounderWalletFinalBeforeMainnet(t *testing.T) {
	policy := DefaultPolicyV1()

	if !policy.FounderWalletFinalBeforeMainnet {
		t.Fatalf("founder wallet must be final before mainnet")
	}
	if policy.FounderWalletChangeAfterMainnet {
		t.Fatalf("founder wallet must not change after mainnet")
	}
}

func TestRejectEarlyUnlock(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.EarlyUnlockForbidden = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("early unlock must fail")
	}
}

func TestRejectWrongMonthlyUnlock(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.MonthlyUnlockQC = 30000

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("wrong monthly unlock must fail")
	}
}

func TestRejectFounderMint(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.DoesNotMintFounderAllocation = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("founder allocation mint permission must fail")
	}
}
