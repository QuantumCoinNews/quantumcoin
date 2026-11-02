package miner

import (
	"strings"
	"testing"

	"quantumcoin/config"
)

func TestResolveMinerEntryDifficultyV1UsesRequestedValue(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)

	got, err := resolveMinerEntryDifficultyV1(
		config.Default(),
		nil,
		17,
	)
	if err != nil {
		t.Fatalf(
			"entry difficulty failed: %v",
			err,
		)
	}

	if got != 17 {
		t.Fatalf(
			"difficulty: got %d, want 17",
			got,
		)
	}
}

func TestResolveMinerEntryDifficultyV1UsesLocalOverride(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)

	t.Setenv(
		"QC_DIFFICULTY_BITS",
		"12",
	)

	got, err := resolveMinerEntryDifficultyV1(
		config.Default(),
		nil,
		17,
	)
	if err != nil {
		t.Fatalf(
			"entry difficulty override failed: %v",
			err,
		)
	}

	if got != 12 {
		t.Fatalf(
			"difficulty: got %d, want 12",
			got,
		)
	}
}

func TestResolveMinerEntryDifficultyV1AcceptsTestnet(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)
	setMinerRuntimeTestnetEnvV1(t)

	got, err := resolveMinerEntryDifficultyV1(
		minerRuntimeTestnetConfigV1(),
		nil,
		8,
	)
	if err != nil {
		t.Fatalf(
			"testnet entry difficulty failed: %v",
			err,
		)
	}

	if got != 8 {
		t.Fatalf(
			"testnet difficulty: got %d, want 8",
			got,
		)
	}
}

func TestResolveQCLocalChainPathV1FallsBackToConfig(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)

	cfg := config.Default()
	cfg.ChainFile =
		config.QCMainnetChainFileV1

	got, err := resolveQCLocalChainPathV1(
		cfg,
		"",
	)
	if err != nil {
		t.Fatalf(
			"config chain fallback failed: %v",
			err,
		)
	}

	if got != config.QCMainnetChainFileV1 {
		t.Fatalf(
			"chain path: got %q, want %q",
			got,
			config.QCMainnetChainFileV1,
		)
	}
}

func TestResolveQCLocalChainPathV1UsesExplicitOverride(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)

	cfg := config.Default()
	cfg.ChainFile = ""

	got, err := resolveQCLocalChainPathV1(
		cfg,
		"  "+config.QCMainnetChainFileV1+"  ",
	)
	if err != nil {
		t.Fatalf(
			"explicit chain override failed: %v",
			err,
		)
	}

	if got != config.QCMainnetChainFileV1 {
		t.Fatalf(
			"chain override: got %q, want %q",
			got,
			config.QCMainnetChainFileV1,
		)
	}
}

func TestResolveQCLocalChainPathV1RejectsEmptyPath(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)

	cfg := config.Default()
	cfg.ChainFile = ""

	_, err := resolveQCLocalChainPathV1(
		cfg,
		"  ",
	)
	if err == nil {
		t.Fatal(
			"empty chain path was accepted",
		)
	}
}

func TestResolveQCLocalChainPathV1RejectsMainnetPathOnTestnet(
	t *testing.T,
) {
	clearMinerRuntimeDifficultyEnvV1(t)
	setMinerRuntimeTestnetEnvV1(t)

	cfg := minerRuntimeTestnetConfigV1()

	_, err := resolveQCLocalChainPathV1(
		cfg,
		config.QCMainnetChainFileV1,
	)
	if err == nil {
		t.Fatal(
			"mainnet chain path was accepted on testnet",
		)
	}

	if !strings.Contains(
		err.Error(),
		"mode guard",
	) {
		t.Fatalf(
			"unexpected chain path error: %v",
			err,
		)
	}
}
