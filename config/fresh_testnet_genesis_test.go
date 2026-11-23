package config

import "testing"

func TestFreshTestnetRejectsHiddenPrefundAndConsensusDrift(t *testing.T) {
	if err := ValidateFreshTestnetGenesisInputs(Default()); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.PremineAddress = "not-a-real-wallet" }, func(c *Config) { c.PremineLiquidAddress = "not-a-real-wallet" }, func(c *Config) { c.PremineLockedAddress = "not-a-real-wallet" },
		func(c *Config) { c.TotalSupply++ }, func(c *Config) { c.InitialReward++ }, func(c *Config) { c.GenesisUnix++ }, func(c *Config) { c.CoinbaseMaturity = 0 }, func(c *Config) { c.TargetBlockTimeSecs++ }, func(c *Config) { c.HalvingPeriodBlocks++ },
	} {
		c := Default()
		change(c)
		if ValidateFreshTestnetGenesisInputs(c) == nil {
			t.Fatal("official testnet accepted hidden allocation or consensus drift")
		}
	}
}
