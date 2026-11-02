package config

import "testing"

// TestQCImmutableEconomyDefaultsV1Locked protects only consensus/economy
// defaults. Runtime-specific ports and development difficulty overrides are
// intentionally tested separately.
func TestQCImmutableEconomyDefaultsV1Locked(t *testing.T) {
	c := Default()

	if c.Symbol != "QC" {
		t.Fatalf("symbol changed: got %q", c.Symbol)
	}
	if c.InitialReward != 50 {
		t.Fatalf("initial reward changed: got %d", c.InitialReward)
	}
	if c.TotalSupply != 25_500_000 {
		t.Fatalf("total supply changed: got %d", c.TotalSupply)
	}
	if c.GenesisUnix != 1_725_158_400 {
		t.Fatalf("genesis unix changed: got %d", c.GenesisUnix)
	}
	if c.HalvingIntervalSecs != int64(63_072_000) {
		t.Fatalf(
			"halving interval changed: got %d",
			c.HalvingIntervalSecs,
		)
	}
	if c.MiningPeriodSecs != int64(315_360_000) {
		t.Fatalf(
			"mining period changed: got %d",
			c.MiningPeriodSecs,
		)
	}
	if c.CoinbaseMaturity != 10 {
		t.Fatalf(
			"coinbase maturity changed: got %d",
			c.CoinbaseMaturity,
		)
	}
	if c.PreminePercent != 5 {
		t.Fatalf(
			"founder total percent changed: got %d",
			c.PreminePercent,
		)
	}
	if c.PremineLiquidPercent != 1 {
		t.Fatalf(
			"founder liquid percent changed: got %d",
			c.PremineLiquidPercent,
		)
	}
	if c.PremineLockedPercent != 4 {
		t.Fatalf(
			"founder locked percent changed: got %d",
			c.PremineLockedPercent,
		)
	}
	if c.FounderVestingCliffMonths != 12 {
		t.Fatalf(
			"founder cliff changed: got %d",
			c.FounderVestingCliffMonths,
		)
	}
	if c.FounderVestingDurationMonths != 48 {
		t.Fatalf(
			"founder vesting duration changed: got %d",
			c.FounderVestingDurationMonths,
		)
	}
	if c.MiningPoolAddress != QCMainnetImmutableMiningPoolAddressV1 {
		t.Fatalf(
			"mining pool address changed: got %q want %q",
			c.MiningPoolAddress,
			QCMainnetImmutableMiningPoolAddressV1,
		)
	}
	if c.BurnAddress != QCMainnetImmutableBurnAddressV1 {
		t.Fatalf(
			"burn address changed: got %q want %q",
			c.BurnAddress,
			QCMainnetImmutableBurnAddressV1,
		)
	}
}
