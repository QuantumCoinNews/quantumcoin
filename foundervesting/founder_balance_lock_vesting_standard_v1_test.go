package foundervesting

import "testing"

func TestFounderBalanceLockVestingStandardV1Locked(t *testing.T) {
	if MaxSupplyQCV1 != 25_500_000 {
		t.Fatalf("max supply changed: got %d", MaxSupplyQCV1)
	}
	if FounderTotalQCV1 != 1_275_000 {
		t.Fatalf("founder total QC changed: got %d", FounderTotalQCV1)
	}
	if FounderLiquidQCV1 != 255_000 {
		t.Fatalf("founder liquid QC changed: got %d", FounderLiquidQCV1)
	}
	if FounderLockedQCV1 != 1_020_000 {
		t.Fatalf("founder locked QC changed: got %d", FounderLockedQCV1)
	}
	if FounderTotalPercentV1 != 5 {
		t.Fatalf("founder total percent changed: got %d", FounderTotalPercentV1)
	}
	if FounderLiquidPercentV1 != 1 {
		t.Fatalf("founder liquid percent changed: got %d", FounderLiquidPercentV1)
	}
	if FounderLockedPercentV1 != 4 {
		t.Fatalf("founder locked percent changed: got %d", FounderLockedPercentV1)
	}
	if CliffMonthsV1 != 12 {
		t.Fatalf("cliff months changed: got %d", CliffMonthsV1)
	}
	if VestingMonthsV1 != 48 {
		t.Fatalf("vesting months changed: got %d", VestingMonthsV1)
	}
	if TotalScheduleMonthsV1 != 60 {
		t.Fatalf("total schedule months changed: got %d", TotalScheduleMonthsV1)
	}
	if MonthlyUnlockQCV1 != 21_250 {
		t.Fatalf("monthly unlock QC changed: got %d", MonthlyUnlockQCV1)
	}

	p := DefaultPolicyV1()
	if p.UnlockStartsAtMonth != 13 {
		t.Fatalf("unlock start month changed: got %d", p.UnlockStartsAtMonth)
	}
	if p.UnlockEndsAtMonth != 60 {
		t.Fatalf("unlock end month changed: got %d", p.UnlockEndsAtMonth)
	}
	if !p.EarlyUnlockForbidden || !p.AdminEarlyUnlockForbidden || !p.AICannotUnlock || !p.GameCannotUnlock || !p.TGWTCannotUnlock || !p.NFTCannotUnlock {
		t.Fatalf("founder lock safety flags changed")
	}
	if !p.FounderAllocationFromFixedSupplyOnly || !p.MaxSupplyAlwaysUnchanged || !p.DoesNotMintFounderAllocation {
		t.Fatalf("fixed supply founder safety flags changed")
	}
}
