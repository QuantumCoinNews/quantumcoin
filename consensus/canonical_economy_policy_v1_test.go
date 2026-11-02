package consensus

import "testing"

func TestCanonicalEconomyPolicyValidV1(
	t *testing.T,
) {
	if err := ValidateCanonicalEconomyPolicyV1(); err != nil {
		t.Fatalf(
			"canonical economy policy rejected: %v",
			err,
		)
	}
}

func TestCanonicalEconomyValuesV1(
	t *testing.T,
) {
	if CanonicalTotalSupplyQCV1 != 25_500_000 {
		t.Fatalf(
			"total supply changed: %d",
			CanonicalTotalSupplyQCV1,
		)
	}

	if CanonicalInitialMiningRewardQCV1 != 50 {
		t.Fatalf(
			"initial mining reward changed: %d",
			CanonicalInitialMiningRewardQCV1,
		)
	}

	if CanonicalCoinbaseMaturityBlocksV1 != 10 {
		t.Fatalf(
			"coinbase maturity changed: %d",
			CanonicalCoinbaseMaturityBlocksV1,
		)
	}

	if CanonicalFounderTotalQCV1 != 1_275_000 ||
		CanonicalFounderLiquidQCV1 != 255_000 ||
		CanonicalFounderLockedQCV1 != 1_020_000 {
		t.Fatalf(
			"founder allocation changed: total=%d liquid=%d locked=%d",
			CanonicalFounderTotalQCV1,
			CanonicalFounderLiquidQCV1,
			CanonicalFounderLockedQCV1,
		)
	}

	if CanonicalFounderMonthlyUnlockQCV1 != 21_250 {
		t.Fatalf(
			"founder monthly unlock changed: %d",
			CanonicalFounderMonthlyUnlockQCV1,
		)
	}
}

func TestCanonicalEconomyValuesAreDerivedV1(
	t *testing.T,
) {
	if CanonicalFounderTotalQCV1 !=
		CanonicalFounderLiquidQCV1+
			CanonicalFounderLockedQCV1 {
		t.Fatal(
			"founder total is not derived from liquid and locked allocations",
		)
	}

	if CanonicalFounderMonthlyUnlockQCV1*
		CanonicalFounderVestingMonthsV1 !=
		CanonicalFounderLockedQCV1 {
		t.Fatal(
			"monthly vesting does not produce the locked allocation",
		)
	}

	totalPools :=
		CanonicalFounderTotalQCV1 +
			CanonicalMiningEmissionPoolQCV1

	if totalPools != CanonicalTotalSupplyQCV1 {
		t.Fatalf(
			"canonical pools=%d total supply=%d",
			totalPools,
			CanonicalTotalSupplyQCV1,
		)
	}
}
