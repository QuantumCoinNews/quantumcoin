package main

import (
	"testing"

	"quantumcoin/config"
)

func clearMainRuntimeNetworkEnvV1(t *testing.T) {
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
		"QC_API_LISTEN",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func setMainRuntimeConfigV1(
	t *testing.T,
	c *config.Config,
) {
	t.Helper()

	oldCfg := cfg
	oldBC := bc

	cfg = c
	bc = nil

	t.Cleanup(func() {
		cfg = oldCfg
		bc = oldBC
	})
}

func TestMainRuntimeNetworkPolicyV1LocalDifficultyOverride(
	t *testing.T,
) {
	clearMainRuntimeNetworkEnvV1(t)
	setMainRuntimeConfigV1(t, config.Default())

	t.Setenv("QC_DIFFICULTY_BITS", "12")

	if got := effectiveDifficultyBits(17); got != 12 {
		t.Fatalf(
			"local effective difficulty: got %d, want 12",
			got,
		)
	}
}

func TestMainRuntimeNetworkPolicyV1MainnetOverrideRejected(
	t *testing.T,
) {
	clearMainRuntimeNetworkEnvV1(t)
	setMainRuntimeConfigV1(t, config.Default())

	t.Setenv("QC_NETWORK", "mainnet")
	t.Setenv("QC_NET_MAGIC", config.QCMainnetMagicV1)
	t.Setenv("QC_DIFFICULTY_BITS", "12")

	if got := effectiveDifficultyBits(17); got != 17 {
		t.Fatalf(
			"mainnet override changed difficulty: got %d, want 17",
			got,
		)
	}
}

func TestMainRuntimeNetworkPolicyV1CanonicalHTTPPortWins(
	t *testing.T,
) {
	clearMainRuntimeNetworkEnvV1(t)
	setMainRuntimeConfigV1(t, config.Default())

	t.Setenv("HTTP_PORT", "8081")

	if got := getHTTPPort(); got != ":8082" {
		t.Fatalf(
			"HTTP port: got %q, want :8082",
			got,
		)
	}
}

func TestMainRuntimeNetworkPolicyV1DefaultListenIsLocalhost(
	t *testing.T,
) {
	clearMainRuntimeNetworkEnvV1(t)
	setMainRuntimeConfigV1(t, config.Default())

	if got := getAPIListenAddr(); got != "127.0.0.1:8082" {
		t.Fatalf(
			"default API listen: got %q",
			got,
		)
	}
}

func TestMainRuntimeNetworkPolicyV1MatchingListenAccepted(
	t *testing.T,
) {
	clearMainRuntimeNetworkEnvV1(t)
	setMainRuntimeConfigV1(t, config.Default())

	t.Setenv(
		"QC_API_LISTEN",
		"0.0.0.0:8082",
	)

	if got := getAPIListenAddr(); got != "0.0.0.0:8082" {
		t.Fatalf(
			"matching API listen: got %q",
			got,
		)
	}
}

func TestMainRuntimeNetworkPolicyV1DifferentListenPortRejected(
	t *testing.T,
) {
	clearMainRuntimeNetworkEnvV1(t)
	setMainRuntimeConfigV1(t, config.Default())

	t.Setenv(
		"QC_API_LISTEN",
		"0.0.0.0:18082",
	)

	if got := getAPIListenAddr(); got != "127.0.0.1:8082" {
		t.Fatalf(
			"mismatched API listen accepted: got %q",
			got,
		)
	}
}

func TestMainRuntimeNetworkPolicyV1MalformedListenRejected(
	t *testing.T,
) {
	clearMainRuntimeNetworkEnvV1(t)
	setMainRuntimeConfigV1(t, config.Default())

	t.Setenv(
		"QC_API_LISTEN",
		"0.0.0.0",
	)

	if got := getAPIListenAddr(); got != "127.0.0.1:8082" {
		t.Fatalf(
			"malformed API listen accepted: got %q",
			got,
		)
	}
}
