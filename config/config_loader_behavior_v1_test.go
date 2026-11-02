package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clearConfigLoaderEnvV1(t *testing.T) {
	t.Helper()

	keys := []string{
		"QC_SYMBOL",
		"QC_INITIAL_REWARD",
		"QC_TOTAL_SUPPLY",
		"QC_GENESIS_UNIX",
		"QC_HALVING_INTERVAL_SECS",
		"QC_MINING_PERIOD_SECS",
		"QC_TARGET_BLOCK_TIME_SECS",
		"QC_HALVING_PERIOD_BLOCKS",
		"QC_DEFAULT_DIFFICULTY_BITS",
		"QC_COINBASE_MATURITY",
		"QC_REWARD_PCT_MINER",
		"QC_REWARD_PCT_STAKE",
		"QC_REWARD_PCT_DEV",
		"QC_REWARD_PCT_BURN",
		"QC_PREMINE_PERCENT",
		"QC_PREMINE_LIQUID_PERCENT",
		"QC_PREMINE_LOCKED_PERCENT",
		"QC_FOUNDER_VESTING_CLIFF_MONTHS",
		"QC_FOUNDER_VESTING_DURATION_MONTHS",
		"QC_HTTP_PORT",
		"QC_P2P_PORT",
		"QC_CHAIN_FILE",
		"QC_BONUS_FILE",
		"QC_WALLET_FILE",
		"QC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_NETWORK_MODE",
		"QC_MODE",
		"QC_NETWORK",
		"QC_CHAIN_MODE",
		"QC_NET_MAGIC",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func writeConfigFixtureV1(
	t *testing.T,
	body string,
) string {
	t.Helper()

	path := filepath.Join(
		t.TempDir(),
		"config.json",
	)

	if err := os.WriteFile(
		path,
		[]byte(body),
		0o600,
	); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	return path
}

func TestConfigLoaderFileThenEnvPrecedenceV1(
	t *testing.T,
) {
	clearConfigLoaderEnvV1(t)

	path := writeConfigFixtureV1(
		t,
		`{
"default_difficulty_bits": 11,
"http_port": ":9191",
"p2p_port": ":3001"
}`,
	)

	t.Setenv(
		"QC_DEFAULT_DIFFICULTY_BITS",
		"13",
	)
	t.Setenv(
		"QC_HTTP_PORT",
		":9292",
	)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.DefaultDifficultyBits != 13 {
		t.Fatalf(
			"ENV difficulty did not win: got %d",
			cfg.DefaultDifficultyBits,
		)
	}

	if cfg.HTTPPort != ":9292" {
		t.Fatalf(
			"ENV HTTP port did not win: got %q",
			cfg.HTTPPort,
		)
	}
}

func TestConfigLoaderRejectsUnknownJSONFieldV1(
	t *testing.T,
) {
	clearConfigLoaderEnvV1(t)

	path := writeConfigFixtureV1(
		t,
		`{
"default_dificulty_bits": 8
}`,
	)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected unknown JSON field rejection")
	}

	if !strings.Contains(
		err.Error(),
		"unknown field",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigLoaderRejectsLegacyPascalCaseV1(
	t *testing.T,
) {
	clearConfigLoaderEnvV1(t)

	path := writeConfigFixtureV1(
		t,
		`{
"HTTPPort": ":6161"
}`,
	)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected PascalCase rejection")
	}

	if !strings.Contains(
		err.Error(),
		"unknown field",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigLoaderAppliesExplicitZeroValuesV1(
	t *testing.T,
) {
	clearConfigLoaderEnvV1(t)

	// Coinbase maturity is immutable on mainnet. Use an explicitly
	// isolated testnet chain to continue testing JSON zero semantics.
	t.Setenv("QC_TESTNET", "1")

	path := writeConfigFixtureV1(
		t,
		`{
"chain_file": "chain_data_testnet.dat",
"coinbase_maturity": 0,
"reward_pct_stake": 0,
"reward_pct_dev": 0,
"reward_pct_burn": 0
}`,
	)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.CoinbaseMaturity != 0 {
		t.Fatalf(
			"coinbase maturity zero lost: got %d",
			cfg.CoinbaseMaturity,
		)
	}

	if cfg.RewardPctStake != 0 {
		t.Fatalf(
			"stake zero lost: got %d",
			cfg.RewardPctStake,
		)
	}

	if cfg.RewardPctDev != 0 {
		t.Fatalf(
			"dev zero lost: got %d",
			cfg.RewardPctDev,
		)
	}

	if cfg.RewardPctBurn != 0 {
		t.Fatalf(
			"burn zero lost: got %d",
			cfg.RewardPctBurn,
		)
	}
}
func TestConfigLoaderReloadsEveryCallV1(
	t *testing.T,
) {
	clearConfigLoaderEnvV1(t)

	firstPath := writeConfigFixtureV1(
		t,
		`{
"default_difficulty_bits": 9,
"http_port": ":8181",
"p2p_port": ":3001"
}`,
	)

	secondPath := filepath.Join(
		t.TempDir(),
		"second.json",
	)

	if err := os.WriteFile(
		secondPath,
		[]byte(`{
"default_difficulty_bits": 10,
"http_port": ":8282",
"p2p_port": ":3001"
}`),
		0o600,
	); err != nil {
		t.Fatalf("write second config: %v", err)
	}

	first, err := Load(firstPath)
	if err != nil {
		t.Fatalf("first Load failed: %v", err)
	}

	second, err := Load(secondPath)
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}

	if first.DefaultDifficultyBits != 9 {
		t.Fatalf(
			"first difficulty mismatch: got %d",
			first.DefaultDifficultyBits,
		)
	}

	if second.DefaultDifficultyBits != 10 {
		t.Fatalf(
			"second config was not loaded: got %d",
			second.DefaultDifficultyBits,
		)
	}

	if second.HTTPPort != ":8282" {
		t.Fatalf(
			"second HTTP port mismatch: got %q",
			second.HTTPPort,
		)
	}
}

func TestRootConfigCanonicalSnakeCaseV1(
	t *testing.T,
) {
	clearConfigLoaderEnvV1(t)

	path := filepath.Join(
		"..",
		"config.json",
	)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf(
			"root config strict load failed: %v",
			err,
		)
	}

	if cfg.DefaultDifficultyBits != 16 {
		t.Fatalf(
			"root difficulty mismatch: got %d",
			cfg.DefaultDifficultyBits,
		)
	}

	if cfg.HTTPPort != ":8082" {
		t.Fatalf(
			"root HTTP port mismatch: got %q",
			cfg.HTTPPort,
		)
	}

	if cfg.PreminePercent != 5 ||
		cfg.PremineLiquidPercent != 1 ||
		cfg.PremineLockedPercent != 4 {
		t.Fatalf(
			"founder percentages mismatch: %d/%d/%d",
			cfg.PreminePercent,
			cfg.PremineLiquidPercent,
			cfg.PremineLockedPercent,
		)
	}

	if cfg.FounderVestingCliffMonths != 12 {
		t.Fatalf(
			"founder cliff mismatch: got %d",
			cfg.FounderVestingCliffMonths,
		)
	}

	if cfg.FounderVestingDurationMonths != 48 {
		t.Fatalf(
			"founder duration mismatch: got %d",
			cfg.FounderVestingDurationMonths,
		)
	}
}
