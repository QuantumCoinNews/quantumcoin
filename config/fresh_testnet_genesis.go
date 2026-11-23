package config

import "fmt"

// ValidateFreshTestnetGenesisInputs pins the official fresh testnet's monetary
// inputs without changing mainnet policy. Test QC is mined, never bonus-minted.
func ValidateFreshTestnetGenesisInputs(c *Config) error {
	if c == nil {
		return fmt.Errorf("fresh testnet config required")
	}
	expected := Default()
	if c.PremineAddress != "" || c.PremineLiquidAddress != "" || c.PremineLockedAddress != "" {
		return fmt.Errorf("fresh testnet genesis has no prefunded bonus/founder wallets; fund test distributions from mined QC")
	}
	if c.TotalSupply != expected.TotalSupply || c.InitialReward != expected.InitialReward || c.GenesisUnix != expected.GenesisUnix || c.CoinbaseMaturity != expected.CoinbaseMaturity || c.TargetBlockTimeSecs != expected.TargetBlockTimeSecs || c.HalvingIntervalSecs != expected.HalvingIntervalSecs || c.HalvingPeriodBlocks != expected.HalvingPeriodBlocks || c.MiningPeriodSecs != expected.MiningPeriodSecs {
		return fmt.Errorf("fresh testnet consensus inputs differ from the canonical profile")
	}
	return nil
}
