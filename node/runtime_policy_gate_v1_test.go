package node

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func clearRuntimePolicyNetworkEnvironmentV1(
	t *testing.T,
) {
	t.Helper()

	for _, key := range []string{
		"QC_TESTNET",
		"QC_PUBLIC_TESTNET",
		"QC_API_PUBLIC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_MAINNET",
		"QC_NETWORK",
		"QC_NETWORK_MODE",
		"QC_CHAIN_MODE",
		"QC_MODE",
		"QC_NET_MAGIC",
		"QC_DIFFICULTY_BITS",
		"HTTP_PORT",
	} {
		t.Setenv(key, "")
	}
}

func defaultRuntimePolicyConfigV1(
	t *testing.T,
) *config.Config {
	t.Helper()
	clearRuntimePolicyNetworkEnvironmentV1(t)

	cfg := config.Default()
	cfg.ChainFile = filepath.Join(
		t.TempDir(),
		config.QCMainnetChainFileV1,
	)

	return cfg
}

func TestDefaultRuntimePolicyGateValidV1(
	t *testing.T,
) {
	cfg := defaultRuntimePolicyConfigV1(t)

	if err := ValidateRuntimePolicyGateV1(
		cfg,
		DefaultRuntimePolicySetV1(),
	); err != nil {
		t.Fatalf(
			"default runtime policy gate rejected: %v",
			err,
		)
	}
}

func TestRuntimePolicyGateRejectsInvalidPackagePoliciesV1(
	t *testing.T,
) {
	tests := []struct {
		name   string
		mutate func(*RuntimePolicySetV1)
	}{
		{
			name: "burn",
			mutate: func(policies *RuntimePolicySetV1) {
				policies.Burn.AutoBurnForbidden = false
			},
		},
		{
			name: "economy",
			mutate: func(policies *RuntimePolicySetV1) {
				policies.Economy.TotalSupplyQC++
			},
		},
		{
			name: "founder-vesting",
			mutate: func(policies *RuntimePolicySetV1) {
				policies.FounderVesting.MonthlyUnlockQC++
			},
		},
		{
			name: "hardfork",
			mutate: func(policies *RuntimePolicySetV1) {
				policies.Hardfork.HardforkExecutionNow = true
			},
		},
		{
			name: "network-fee",
			mutate: func(policies *RuntimePolicySetV1) {
				policies.NetworkFee.FeeIsNewMint = true
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := defaultRuntimePolicyConfigV1(t)
			policies := DefaultRuntimePolicySetV1()
			test.mutate(&policies)

			if err := ValidateRuntimePolicyGateV1(
				cfg,
				policies,
			); err == nil {
				t.Fatalf(
					"invalid %s policy was accepted",
					test.name,
				)
			}
		})
	}
}

func TestRuntimePolicyGateRejectsCanonicalAddressMismatchV1(
	t *testing.T,
) {
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.BurnAddress = "QC_DIFFERENT_BURN_SINK"

	err := ValidateRuntimePolicyGateV1(
		cfg,
		DefaultRuntimePolicySetV1(),
	)
	if err == nil {
		t.Fatal(
			"canonical burn address mismatch was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"mainnet immutable field burn_address mismatch",
	) {
		t.Fatalf(
			"unexpected mismatch error: %v",
			err,
		)
	}
}

func TestRuntimePolicyGateRejectsCanonicalPoolMismatchV1(
	t *testing.T,
) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
		want   string
	}{
		{
			name: "mining",
			mutate: func(cfg *config.Config) {
				cfg.MiningPoolAddress = "QC_DIFFERENT_MINING_POOL"
			},
			want: "mainnet immutable field mining_pool_address mismatch",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := defaultRuntimePolicyConfigV1(t)
			testCase.mutate(cfg)

			err := ValidateRuntimePolicyGateV1(
				cfg,
				DefaultRuntimePolicySetV1(),
			)
			if err == nil {
				t.Fatal(
					"canonical economy pool mismatch was accepted",
				)
			}

			if !strings.Contains(
				err.Error(),
				testCase.want,
			) {
				t.Fatalf(
					"unexpected pool mismatch error: %v",
					err,
				)
			}
		})
	}
}

func TestRuntimePolicyGateBlocksExplicitMainnetV1(
	t *testing.T,
) {
	cfg := defaultRuntimePolicyConfigV1(t)

	t.Setenv("QC_MAINNET", "1")

	liquidWallet := wallet.NewWallet()
	lockedWallet := wallet.NewWallet()

	cfg.PremineLiquidAddress =
		liquidWallet.GetAddress()
	cfg.PremineLockedAddress =
		lockedWallet.GetAddress()

	err := ValidateRuntimePolicyGateV1(
		cfg,
		DefaultRuntimePolicySetV1(),
	)
	if err == nil {
		t.Fatal(
			"explicit mainnet started without runtime readiness",
		)
	}

	if !strings.Contains(
		err.Error(),
		"explicit mainnet startup blocked",
	) {
		t.Fatalf(
			"unexpected explicit mainnet error: %v",
			err,
		)
	}
}

func TestBootstrapWithConfigAppliesRuntimePolicyGateV1(
	t *testing.T,
) {
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.BurnAddress = "QC_POLICY_GATE_BYPASS"

	_, err := BootstrapWithConfig(
		cfg,
		false,
	)
	if err == nil {
		t.Fatal(
			"BootstrapWithConfig bypassed runtime policy gate",
		)
	}

	if !strings.Contains(
		err.Error(),
		"runtime policy gate",
	) {
		t.Fatalf(
			"unexpected bootstrap error: %v",
			err,
		)
	}

	if _, statErr := os.Stat(
		cfg.ChainFile,
	); !os.IsNotExist(statErr) {
		t.Fatalf(
			"chain storage was touched before policy rejection: %v",
			statErr,
		)
	}
}

func TestRuntimePolicyGateExplicitMainnetNoLongerBlockedByBurnReadinessV1(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)

	t.Setenv("QC_MAINNET", "1")

	liquidWallet := wallet.NewWallet()
	lockedWallet := wallet.NewWallet()

	cfg.PremineLiquidAddress = liquidWallet.GetAddress()
	cfg.PremineLockedAddress = lockedWallet.GetAddress()

	err := ValidateRuntimePolicyGateV1(
		cfg,
		DefaultRuntimePolicySetV1(),
	)
	if err == nil {
		t.Fatal("explicit mainnet unexpectedly started while network-fee readiness blocker remains")
	}

	message := err.Error()

	if strings.Contains(
		message,
		"cryptographically unspendable burn sink proof",
	) {
		t.Fatalf(
			"completed burn readiness still blocks explicit mainnet: %v",
			err,
		)
	}

	if !strings.Contains(
		message,
		"enforceable network fee denomination is not finalized",
	) {
		t.Fatalf(
			"remaining network-fee readiness blocker missing: %v",
			err,
		)
	}
}
