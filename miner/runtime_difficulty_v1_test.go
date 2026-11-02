package miner

import (
	"strings"
	"testing"

	"quantumcoin/config"
)

func clearMinerRuntimeDifficultyEnvV1(
	t *testing.T,
) {
	t.Helper()

	keys := []string{
		"QC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_API_PUBLIC_TESTNET",
		"QC_PUBLIC_TESTNET",
		"QC_MAINNET",
		"QC_NET_MAGIC",
		"QC_NETWORK",
		"QC_NETWORK_MODE",
		"QC_CHAIN_MODE",
		"QC_MODE",
		"QC_DIFFICULTY_BITS",
		"QC_HTTP_PORT",
		"QC_P2P_PORT",
		"HTTP_PORT",
		"QC_API_LISTEN",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func minerRuntimeTestnetConfigV1() *config.Config {
	cfg := config.Default()

	cfg.ChainFile = config.QCTestnetChainFileV1
	cfg.HTTPPort = ":18082"
	cfg.P2PPort = ":13001"
	cfg.DefaultDifficultyBits = 8

	return cfg
}

func setMinerRuntimeTestnetEnvV1(
	t *testing.T,
) {
	t.Helper()

	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_NETWORK", "testnet")
	t.Setenv("QC_CHAIN_MODE", "testnet")
	t.Setenv(
		"QC_NET_MAGIC",
		config.QCTestnetMagicV1,
	)
}

func TestResolveRuntimeDifficultyV1UsesRequestedDynamicValue(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)

	got, err := resolveRuntimeDifficultyV1(
		config.Default(),
		nil,
		17,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if got != 17 {
		t.Fatalf(
			"difficulty: got %d, want 17",
			got,
		)
	}
}

func TestResolveRuntimeDifficultyV1UsesBaseWithoutBlockchain(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)

	got, err := resolveRuntimeDifficultyV1(
		config.Default(),
		nil,
		0,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if got != 16 {
		t.Fatalf(
			"difficulty: got %d, want 16",
			got,
		)
	}
}

func TestResolveRuntimeDifficultyV1AllowsLocalReleaseOverride(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)
	t.Setenv("QC_DIFFICULTY_BITS", "12")

	got, err := resolveRuntimeDifficultyV1(
		config.Default(),
		nil,
		17,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if got != 12 {
		t.Fatalf(
			"difficulty: got %d, want 12",
			got,
		)
	}
}

func TestResolveRuntimeDifficultyV1AcceptsCleanTestnet(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)
	setMinerRuntimeTestnetEnvV1(t)

	got, err := resolveRuntimeDifficultyV1(
		minerRuntimeTestnetConfigV1(),
		nil,
		8,
	)
	if err != nil {
		t.Fatalf("testnet resolve failed: %v", err)
	}

	if got != 8 {
		t.Fatalf(
			"testnet difficulty: got %d, want 8",
			got,
		)
	}
}

func TestResolveRuntimeDifficultyV1RejectsTestnetOverride(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)
	setMinerRuntimeTestnetEnvV1(t)

	t.Setenv("QC_DIFFICULTY_BITS", "12")

	_, err := resolveRuntimeDifficultyV1(
		minerRuntimeTestnetConfigV1(),
		nil,
		8,
	)
	if err == nil {
		t.Fatal("testnet fixed override was accepted")
	}

	if !strings.Contains(err.Error(), "forbids") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRuntimeDifficultyV1RejectsMainnetOverride(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)

	t.Setenv("QC_NETWORK", "mainnet")
	t.Setenv(
		"QC_NET_MAGIC",
		config.QCMainnetMagicV1,
	)
	t.Setenv("QC_DIFFICULTY_BITS", "12")

	_, err := resolveRuntimeDifficultyV1(
		config.Default(),
		nil,
		16,
	)
	if err == nil {
		t.Fatal("mainnet fixed override was accepted")
	}

	if !strings.Contains(err.Error(), "forbids") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRuntimeDifficultyV1RejectsNilConfig(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)

	_, err := resolveRuntimeDifficultyV1(
		nil,
		nil,
		16,
	)
	if err == nil {
		t.Fatal("nil config was accepted")
	}
}
