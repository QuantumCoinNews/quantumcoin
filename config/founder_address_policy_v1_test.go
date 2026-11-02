package config_test

import (
	"testing"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func setExplicitMainnetFounderPolicyEnvV1(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"QC_TESTNET",
		"QC_PUBLIC_TESTNET",
		"QC_API_PUBLIC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_NETWORK",
		"QC_NETWORK_MODE",
		"QC_CHAIN_MODE",
		"QC_MODE",
		"QC_NET_MAGIC",
	} {
		t.Setenv(key, "")
	}

	t.Setenv("QC_MAINNET", "1")
}

func validFounderConfigV1(t *testing.T) *config.Config {
	t.Helper()

	liquid := wallet.NewWallet()
	locked := wallet.NewWallet()

	if liquid.GetAddress() == locked.GetAddress() {
		t.Fatalf("generated founder addresses are unexpectedly equal")
	}

	cfg := config.Default()
	cfg.PremineLiquidAddress = liquid.GetAddress()
	cfg.PremineLockedAddress = locked.GetAddress()

	return cfg
}

func TestFounderAddressPolicyAcceptsDistinctValidMainnetAddressesV1(
	t *testing.T,
) {
	setExplicitMainnetFounderPolicyEnvV1(t)
	cfg := validFounderConfigV1(t)

	if err := config.ValidateFounderAddressBindingV1(cfg); err != nil {
		t.Fatalf("valid founder binding rejected: %v", err)
	}
}

func TestFounderAddressPolicyRejectsMissingMainnetAddressesV1(
	t *testing.T,
) {
	setExplicitMainnetFounderPolicyEnvV1(t)
	cfg := config.Default()

	if err := config.ValidateFounderAddressBindingV1(cfg); err == nil {
		t.Fatalf("expected missing founder address rejection")
	}
}

func TestFounderAddressPolicyRejectsSameMainnetAddressV1(
	t *testing.T,
) {
	setExplicitMainnetFounderPolicyEnvV1(t)
	cfg := validFounderConfigV1(t)
	cfg.PremineLockedAddress = cfg.PremineLiquidAddress

	if err := config.ValidateFounderAddressBindingV1(cfg); err == nil {
		t.Fatalf("expected same founder address rejection")
	}
}

func TestFounderAddressPolicyRejectsInvalidMainnetAddressV1(
	t *testing.T,
) {
	setExplicitMainnetFounderPolicyEnvV1(t)
	cfg := validFounderConfigV1(t)
	cfg.PremineLockedAddress = "not-a-valid-qc-address"

	if err := config.ValidateFounderAddressBindingV1(cfg); err == nil {
		t.Fatalf("expected invalid founder address rejection")
	}
}

func TestFounderAddressPolicyAllowsUnboundLocalReleaseV1(
	t *testing.T,
) {
	for _, key := range []string{
		"QC_MAINNET",
		"QC_TESTNET",
		"QC_PUBLIC_TESTNET",
		"QC_API_PUBLIC_TESTNET",
		"QC_TESTNET_PUBLIC_API",
		"QC_NETWORK",
		"QC_NETWORK_MODE",
		"QC_CHAIN_MODE",
		"QC_MODE",
		"QC_NET_MAGIC",
	} {
		t.Setenv(key, "")
	}

	cfg := config.Default()
	if err := config.ValidateFounderAddressBindingV1(cfg); err != nil {
		t.Fatalf(
			"local release should not require final founder binding: %v",
			err,
		)
	}
}
