package config

import (
	"errors"
	"testing"
)

func clearTestnetLabelEnvV1(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"QC_TESTNET",
		"QC_PUBLIC_TESTNET",
		"QC_API_PUBLIC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_NET_MAGIC",
		"QC_NETWORK",
		"QC_NETWORK_MODE",
		"QC_CHAIN_MODE",
		"QC_MODE",
	} {
		t.Setenv(name, "")
	}
}

func TestGetNetworkLabelInfoMainnetV1(t *testing.T) {
	clearTestnetLabelEnvV1(t)

	info := GetNetworkLabelInfoV1(QCMainnetChainFileV1)
	if info.IsTestnet {
		t.Fatalf("mainnet chain file without testnet env must not be testnet")
	}
	if info.Label != "MAINNET" {
		t.Fatalf("expected MAINNET label, got %q", info.Label)
	}
}

func TestGetNetworkLabelInfoTestnetFromEnvV1(t *testing.T) {
	clearTestnetLabelEnvV1(t)
	t.Setenv("QC_TESTNET", "1")

	info := GetNetworkLabelInfoV1(QCTestnetChainFileV1)
	if !info.IsTestnet {
		t.Fatalf("expected testnet label")
	}
	if info.Label != "TESTNET" || info.Badge != "TESTNET" {
		t.Fatalf("expected TESTNET label/badge, got label=%q badge=%q", info.Label, info.Badge)
	}
}

func TestGetNetworkLabelInfoTestnetFromChainFileV1(t *testing.T) {
	clearTestnetLabelEnvV1(t)

	info := GetNetworkLabelInfoV1("data/testnet/chain_data_testnet.dat")
	if !info.IsTestnet {
		t.Fatalf("testnet chain file must produce testnet label")
	}
}

func TestRequireVisibleTestnetLabelV1(t *testing.T) {
	clearTestnetLabelEnvV1(t)
	t.Setenv("QC_TESTNET", "1")

	if err := RequireVisibleTestnetLabelV1(QCTestnetChainFileV1, "QuantumCoin"); !errors.Is(err, ErrVisibleTestnetLabelRequiredV1) {
		t.Fatalf("expected visible testnet label error, got %v", err)
	}

	if err := RequireVisibleTestnetLabelV1(QCTestnetChainFileV1, "QuantumCoin TESTNET"); err != nil {
		t.Fatalf("visible TESTNET label should pass: %v", err)
	}
}
