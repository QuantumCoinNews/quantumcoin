package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clearMainnetImmutableEnvV1(t *testing.T) {
	t.Helper()

	keys := []string{
		"QC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_PUBLIC_TESTNET",
		"QC_MAINNET",
		"QC_NET_MAGIC",
		"QC_NETWORK",
		"QC_NETWORK_MODE",
		"QC_CHAIN_MODE",
		"QC_MODE",
		"QC_SYMBOL",
		"QC_INITIAL_REWARD",
		"QC_TOTAL_SUPPLY",
		"QC_GENESIS_UNIX",
		"QC_HALVING_INTERVAL_SECS",
		"QC_MINING_PERIOD_SECS",
		"QC_TARGET_BLOCK_TIME_SECS",
		"QC_DEFAULT_DIFFICULTY_BITS",
		"QC_COINBASE_MATURITY",
		"QC_PREMINE_PERCENT",
		"QC_PREMINE_LIQUID_PERCENT",
		"QC_PREMINE_LOCKED_PERCENT",
		"QC_FOUNDER_VESTING_CLIFF_MONTHS",
		"QC_FOUNDER_VESTING_DURATION_MONTHS",
		"QC_BURN_ADDRESS",
		"QC_MINING_POOL_ADDRESS",
		"QC_CHAIN_FILE",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func TestValidateMainnetImmutableConfigV1DefaultPasses(
	t *testing.T,
) {
	clearMainnetImmutableEnvV1(t)

	if err := Default().Validate(); err != nil {
		t.Fatalf("canonical default rejected: %v", err)
	}
}

func TestValidateMainnetImmutableConfigV1RejectsEveryField(
	t *testing.T,
) {
	tests := []struct {
		field  string
		mutate func(*Config)
	}{
		{"symbol", func(c *Config) {
			c.Symbol = "OTHER"
		}},
		{"initial_reward", func(c *Config) {
			c.InitialReward = 51
		}},
		{"total_supply", func(c *Config) {
			c.TotalSupply = 25_500_001
		}},
		{"genesis_unix", func(c *Config) {
			c.GenesisUnix++
		}},
		{"halving_interval_secs", func(c *Config) {
			c.HalvingIntervalSecs++
		}},
		{"mining_period_secs", func(c *Config) {
			c.MiningPeriodSecs++
		}},
		{"coinbase_maturity", func(c *Config) {
			c.CoinbaseMaturity = 11
		}},
		{"premine_percent", func(c *Config) {
			c.PreminePercent = 6
		}},
		{"premine_liquid_percent", func(c *Config) {
			c.PremineLiquidPercent = 2
		}},
		{"premine_locked_percent", func(c *Config) {
			c.PremineLockedPercent = 5
		}},
		{"founder_vesting_cliff_months", func(c *Config) {
			c.FounderVestingCliffMonths = 13
		}},
		{"founder_vesting_duration_months", func(c *Config) {
			c.FounderVestingDurationMonths = 49
		}},
		{"burn_address", func(c *Config) {
			c.BurnAddress = "QC_DIFFERENT_BURN_SINK"
		}},
		{"mining_pool_address", func(c *Config) {
			c.MiningPoolAddress = "QC_DIFFERENT_MINING_POOL"
		}},
	}

	for _, testCase := range tests {
		t.Run(testCase.field, func(t *testing.T) {
			clearMainnetImmutableEnvV1(t)

			cfg := Default()
			testCase.mutate(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatalf(
					"expected %s mainnet rejection",
					testCase.field,
				)
			}

			if !strings.Contains(
				err.Error(),
				testCase.field,
			) {
				t.Fatalf(
					"error does not identify %s: %v",
					testCase.field,
					err,
				)
			}
		})
	}
}

func TestValidateMainnetImmutableConfigV1RejectsExplicitMainnetDifficultyAndTargetTimeV1(
	t *testing.T,
) {
	tests := []struct {
		field  string
		mutate func(*Config)
	}{
		{
			field: "target_block_time_secs",
			mutate: func(c *Config) {
				c.TargetBlockTimeSecs++
			},
		},
		{
			field: "default_difficulty_bits",
			mutate: func(c *Config) {
				c.DefaultDifficultyBits++
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.field, func(t *testing.T) {
			clearMainnetImmutableEnvV1(t)
			t.Setenv("QC_MAINNET", "1")

			cfg := Default()
			testCase.mutate(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatalf(
					"expected explicit mainnet %s rejection",
					testCase.field,
				)
			}

			if !strings.Contains(
				err.Error(),
				testCase.field,
			) {
				t.Fatalf(
					"error does not identify %s: %v",
					testCase.field,
					err,
				)
			}
		})
	}
}
func TestValidateMainnetImmutableConfigV1AllowsTestnetOverride(
	t *testing.T,
) {
	clearMainnetImmutableEnvV1(t)

	t.Setenv("QC_TESTNET", "1")

	cfg := Default()
	cfg.ChainFile = "chain_data_testnet.dat"
	cfg.TotalSupply = 25_500_001

	if err := cfg.Validate(); err != nil {
		t.Fatalf(
			"isolated testnet override rejected: %v",
			err,
		)
	}
}

func TestLoadRejectsMainnetImmutableFileOverrideV1(
	t *testing.T,
) {
	clearMainnetImmutableEnvV1(t)

	path := filepath.Join(
		t.TempDir(),
		"config.json",
	)

	if err := os.WriteFile(
		path,
		[]byte(`{
"total_supply": 25500001
}`),
		0o600,
	); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected mainnet file override rejection")
	}

	if !strings.Contains(err.Error(), "total_supply") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsMainnetImmutableEnvOverrideV1(
	t *testing.T,
) {
	clearMainnetImmutableEnvV1(t)

	t.Setenv("QC_TOTAL_SUPPLY", "25500001")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected mainnet ENV override rejection")
	}

	if !strings.Contains(err.Error(), "total_supply") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetValidatedClosesMainnetOverrideBypassV1(
	t *testing.T,
) {
	clearMainnetImmutableEnvV1(t)

	defer func() {
		_ = SetValidated(Default())
	}()

	if err := SetValidated(Default()); err != nil {
		t.Fatalf("set canonical config: %v", err)
	}

	bad := Default()
	bad.TotalSupply = 25_500_001

	if err := SetValidated(bad); err == nil {
		t.Fatal("SetValidated accepted immutable override")
	}

	Set(bad)

	if got := Current().TotalSupply; got != 25_500_000 {
		t.Fatalf(
			"invalid legacy Set changed total supply: %d",
			got,
		)
	}

	source := Default()

	if err := SetValidated(source); err != nil {
		t.Fatalf("set canonical source: %v", err)
	}

	source.TotalSupply = 1

	if got := Current().TotalSupply; got != 25_500_000 {
		t.Fatalf(
			"caller mutation changed current config: %d",
			got,
		)
	}
}
