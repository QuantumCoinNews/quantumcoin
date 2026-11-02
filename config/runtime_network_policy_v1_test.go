package config

import (
	"strings"
	"testing"
)

func clearRuntimeNetworkPolicyEnvV1(t *testing.T) {
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
		"QC_DIFFICULTY_BITS",
		"HTTP_PORT",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func runtimeTestnetConfigV1() *Config {
	c := Default()
	c.ChainFile = QCTestnetChainFileV1
	c.HTTPPort = ":18082"
	c.P2PPort = ":13001"
	c.DefaultDifficultyBits = 8

	return c
}

func TestValidateRuntimeNetworkProfileV1LocalReleasePasses(
	t *testing.T,
) {
	clearRuntimeNetworkPolicyEnvV1(t)

	if err := ValidateRuntimeNetworkProfileV1(
		Default(),
	); err != nil {
		t.Fatalf("local release rejected: %v", err)
	}
}

func TestResolveEffectiveDifficultyBitsV1UsesDynamicBits(
	t *testing.T,
) {
	clearRuntimeNetworkPolicyEnvV1(t)

	info, err := ResolveEffectiveDifficultyBitsV1(
		Default(),
		17,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if info.EffectiveBits != 17 {
		t.Fatalf(
			"effective bits: got %d, want 17",
			info.EffectiveBits,
		)
	}

	if info.OverrideApplied {
		t.Fatal("unexpected fixed override")
	}
}

func TestResolveEffectiveDifficultyBitsV1AllowsLocalReleaseOverride(
	t *testing.T,
) {
	clearRuntimeNetworkPolicyEnvV1(t)
	t.Setenv("QC_DIFFICULTY_BITS", "12")

	info, err := ResolveEffectiveDifficultyBitsV1(
		Default(),
		17,
	)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	if info.EffectiveBits != 12 {
		t.Fatalf(
			"effective bits: got %d, want 12",
			info.EffectiveBits,
		)
	}

	if !info.OverrideApplied {
		t.Fatal("local release override was not recorded")
	}
}

func TestResolveEffectiveDifficultyBitsV1RejectsTestnetOverride(
	t *testing.T,
) {
	clearRuntimeNetworkPolicyEnvV1(t)

	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_NET_MAGIC", QCTestnetMagicV1)
	t.Setenv("QC_DIFFICULTY_BITS", "12")

	_, err := ResolveEffectiveDifficultyBitsV1(
		runtimeTestnetConfigV1(),
		8,
	)
	if err == nil {
		t.Fatal("testnet accepted fixed difficulty override")
	}

	if !strings.Contains(err.Error(), "forbids") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveEffectiveDifficultyBitsV1RejectsMainnetOverride(
	t *testing.T,
) {
	clearRuntimeNetworkPolicyEnvV1(t)

	t.Setenv("QC_NETWORK", "mainnet")
	t.Setenv("QC_NET_MAGIC", QCMainnetMagicV1)
	t.Setenv("QC_DIFFICULTY_BITS", "12")

	_, err := ResolveEffectiveDifficultyBitsV1(
		Default(),
		16,
	)
	if err == nil {
		t.Fatal("mainnet accepted fixed difficulty override")
	}

	if !strings.Contains(err.Error(), "forbids") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRuntimeNetworkProfileV1TestnetPasses(
	t *testing.T,
) {
	clearRuntimeNetworkPolicyEnvV1(t)

	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_NET_MAGIC", QCTestnetMagicV1)

	if err := ValidateRuntimeNetworkProfileV1(
		runtimeTestnetConfigV1(),
	); err != nil {
		t.Fatalf("testnet rejected: %v", err)
	}
}

func TestValidateRuntimeNetworkProfileV1RejectsProfileMismatch(
	t *testing.T,
) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{
			name: "HTTP port",
			mutate: func(c *Config) {
				c.HTTPPort = ":8081"
			},
			want: "HTTP port",
		},
		{
			name: "P2P port",
			mutate: func(c *Config) {
				c.P2PPort = ":3002"
			},
			want: "P2P port",
		},
		{
			name: "base difficulty",
			mutate: func(c *Config) {
				c.DefaultDifficultyBits = 15
			},
			want: "base difficulty",
		},
		{
			name: "chain file",
			mutate: func(c *Config) {
				c.ChainFile = "other_chain.dat"
			},
			want: "chain file",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			clearRuntimeNetworkPolicyEnvV1(t)

			c := Default()
			testCase.mutate(c)

			err := ValidateRuntimeNetworkProfileV1(c)
			if err == nil {
				t.Fatalf(
					"expected %s mismatch rejection",
					testCase.name,
				)
			}

			if !strings.Contains(
				err.Error(),
				testCase.want,
			) {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}
		})
	}
}

func TestValidateRuntimeNetworkProfileV1RequiresTestnetMagic(
	t *testing.T,
) {
	clearRuntimeNetworkPolicyEnvV1(t)
	t.Setenv("QC_TESTNET", "1")

	err := ValidateRuntimeNetworkProfileV1(
		runtimeTestnetConfigV1(),
	)
	if err == nil {
		t.Fatal("testnet without network magic was accepted")
	}

	if !strings.Contains(err.Error(), "QC_NET_MAGIC") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveEffectiveHTTPPortV1RejectsLegacyConflict(
	t *testing.T,
) {
	clearRuntimeNetworkPolicyEnvV1(t)
	t.Setenv("HTTP_PORT", "8081")

	_, err := ResolveEffectiveHTTPPortV1(Default())
	if err == nil {
		t.Fatal("legacy HTTP_PORT conflict was accepted")
	}

	if !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveEffectiveHTTPPortV1AllowsEquivalentLegacyValue(
	t *testing.T,
) {
	clearRuntimeNetworkPolicyEnvV1(t)
	t.Setenv("HTTP_PORT", "8082")

	got, err := ResolveEffectiveHTTPPortV1(Default())
	if err != nil {
		t.Fatalf("equivalent legacy port rejected: %v", err)
	}

	if got != ":8082" {
		t.Fatalf("HTTP port: got %q, want :8082", got)
	}
}
