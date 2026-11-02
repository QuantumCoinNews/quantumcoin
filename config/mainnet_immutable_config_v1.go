package config

import qconsensus "quantumcoin/consensus"

import (
	"fmt"
	"reflect"
)

const (
	QCMainnetImmutableConfigVersionV1 = "QC_MAINNET_IMMUTABLE_CONFIG_V1"

	QCMainnetImmutableSymbolV1                    = "QC"
	QCMainnetImmutableInitialRewardV1       int   = int(qconsensus.CanonicalInitialMiningRewardQCV1)
	QCMainnetImmutableTotalSupplyV1         int   = int(qconsensus.CanonicalTotalSupplyQCV1)
	QCMainnetImmutableGenesisUnixV1         int64 = 1_725_158_400
	QCMainnetImmutableHalvingIntervalSecsV1 int64 = 63_072_000
	QCMainnetImmutableMiningPeriodSecsV1    int64 = 315_360_000

	QCMainnetImmutableTargetBlockTimeSecsV1   int = 30
	QCMainnetImmutableDefaultDifficultyBitsV1 int = 16
	QCMainnetImmutableCoinbaseMaturityV1      int = qconsensus.CanonicalCoinbaseMaturityBlocksV1
	QCMainnetImmutablePreminePercentV1        int = qconsensus.CanonicalFounderTotalPercentV1
	QCMainnetImmutablePremineLiquidPercentV1  int = qconsensus.CanonicalFounderLiquidPercentV1
	QCMainnetImmutablePremineLockedPercentV1  int = qconsensus.CanonicalFounderLockedPercentV1
	QCMainnetImmutableFounderCliffMonthsV1    int = qconsensus.CanonicalFounderCliffMonthsV1
	QCMainnetImmutableFounderDurationMonthsV1 int = qconsensus.CanonicalFounderVestingMonthsV1

	QCMainnetImmutableMiningPoolAddressV1 = qconsensus.CanonicalMiningEmissionPoolAddressV1
	QCMainnetImmutableBurnAddressV1       = qconsensus.CanonicalBurnSinkAddressV1
)

// ValidateMainnetImmutableConfigV1 prevents JSON, environment,
// admin, AI, game, wallet, miner, and Config.Set paths from
// changing canonical mainnet economy and consensus parameters.
func ValidateMainnetImmutableConfigV1(c *Config) error {
	if c == nil {
		return fmt.Errorf(
			"%s: config cannot be nil",
			QCMainnetImmutableConfigVersionV1,
		)
	}

	// Explicitly isolated testnet configuration is controlled by
	// the separate testnet and chain-file boundary policies.
	if IsTestnetModeEnvV1() {
		return nil
	}

	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"symbol", c.Symbol, QCMainnetImmutableSymbolV1},
		{
			"initial_reward",
			c.InitialReward,
			QCMainnetImmutableInitialRewardV1,
		},
		{
			"total_supply",
			c.TotalSupply,
			QCMainnetImmutableTotalSupplyV1,
		},
		{
			"genesis_unix",
			c.GenesisUnix,
			QCMainnetImmutableGenesisUnixV1,
		},
		{
			"halving_interval_secs",
			c.HalvingIntervalSecs,
			QCMainnetImmutableHalvingIntervalSecsV1,
		},
		{
			"mining_period_secs",
			c.MiningPeriodSecs,
			QCMainnetImmutableMiningPeriodSecsV1,
		},
		{
			"coinbase_maturity",
			c.CoinbaseMaturity,
			QCMainnetImmutableCoinbaseMaturityV1,
		},
		{
			"premine_percent",
			c.PreminePercent,
			QCMainnetImmutablePreminePercentV1,
		},
		{
			"premine_liquid_percent",
			c.PremineLiquidPercent,
			QCMainnetImmutablePremineLiquidPercentV1,
		},
		{
			"premine_locked_percent",
			c.PremineLockedPercent,
			QCMainnetImmutablePremineLockedPercentV1,
		},
		{
			"founder_vesting_cliff_months",
			c.FounderVestingCliffMonths,
			QCMainnetImmutableFounderCliffMonthsV1,
		},
		{
			"founder_vesting_duration_months",
			c.FounderVestingDurationMonths,
			QCMainnetImmutableFounderDurationMonthsV1,
		},
		{"mining_pool_address", c.MiningPoolAddress, QCMainnetImmutableMiningPoolAddressV1},
		{"burn_address", c.BurnAddress, QCMainnetImmutableBurnAddressV1},
	}

	for _, check := range checks {
		if reflect.DeepEqual(check.got, check.want) {
			continue
		}

		return fmt.Errorf(
			"%s: mainnet immutable field %s mismatch: got %v, want %v",
			QCMainnetImmutableConfigVersionV1,
			check.field,
			check.got,
			check.want,
		)
	}

	profile, err := ResolveNetworkProfileV1(c.ChainFile)
	if err != nil {
		return fmt.Errorf(
			"%s: resolve network profile: %w",
			QCMainnetImmutableConfigVersionV1,
			err,
		)
	}

	if profile.IsExplicitMainnet {
		if c.TargetBlockTimeSecs !=
			QCMainnetImmutableTargetBlockTimeSecsV1 {
			return fmt.Errorf(
				"%s: mainnet immutable field target_block_time_secs mismatch: got %v, want %v",
				QCMainnetImmutableConfigVersionV1,
				c.TargetBlockTimeSecs,
				QCMainnetImmutableTargetBlockTimeSecsV1,
			)
		}

		if c.DefaultDifficultyBits !=
			QCMainnetImmutableDefaultDifficultyBitsV1 {
			return fmt.Errorf(
				"%s: mainnet immutable field default_difficulty_bits mismatch: got %v, want %v",
				QCMainnetImmutableConfigVersionV1,
				c.DefaultDifficultyBits,
				QCMainnetImmutableDefaultDifficultyBitsV1,
			)
		}
	}

	return nil
}
