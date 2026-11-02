package config

import "testing"

func clearFaucetPremineEnvV1(t *testing.T) {
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
		"QC_TESTNET_FAUCET_ENABLED",
		"QC_FAUCET_ENABLED",
		"QC_PUBLIC_FAUCET_ENABLED",
		"QC_FAUCET_ADDRESS",
		"QC_FAUCET_PRIVATE_KEY",
		"QC_FAUCET_PRIVATE_KEY_HEX",
		"QC_TESTNET_FAUCET_ADDRESS",
		"QC_TESTNET_FAUCET_PRIVATE_KEY",
		"QC_TESTNET_FAUCET_PRIVATE_KEY_HEX",
		"QC_TESTNET_PREMINE_ENABLED",
		"QC_MAIN_ADDRESS",
		"QC_PREMINE_ADDRESS",
		"QC_PREMINE_LIQUID_ADDRESS",
		"QC_PREMINE_LOCKED_ADDRESS",
		"QC_PREMINE_PERCENT",
		"QC_PREMINE_LIQUID_PERCENT",
		"QC_PREMINE_LOCKED_PERCENT",
	} {
		t.Setenv(name, "")
	}
}

func TestValidateTestnetFaucetPremineSafetyAllowsDisabledSkeletonV1(t *testing.T) {
	clearFaucetPremineEnvV1(t)
	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_TESTNET_FAUCET_ENABLED", "0")
	t.Setenv("QC_TESTNET_PREMINE_ENABLED", "0")
	t.Setenv("QC_PREMINE_PERCENT", "0")

	if err := ValidateTestnetFaucetPremineSafetyV1(); err != nil {
		t.Fatalf("disabled testnet faucet/premine skeleton should be valid: %v", err)
	}
}

func TestValidateTestnetFaucetPremineSafetyRejectsFaucetEnabledV1(t *testing.T) {
	clearFaucetPremineEnvV1(t)
	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_TESTNET_FAUCET_ENABLED", "1")

	if err := ValidateTestnetFaucetPremineSafetyV1(); err == nil {
		t.Fatalf("expected enabled testnet faucet rejection")
	}
}

func TestValidateTestnetFaucetPremineSafetyRejectsFaucetCredentialV1(t *testing.T) {
	clearFaucetPremineEnvV1(t)
	t.Setenv("QC_NET_MAGIC", QCTestnetMagicV1)
	t.Setenv("QC_TESTNET_FAUCET_PRIVATE_KEY_HEX", "not-real")

	if err := ValidateTestnetFaucetPremineSafetyV1(); err == nil {
		t.Fatalf("expected testnet faucet private key rejection")
	}
}

func TestValidateTestnetFaucetPremineSafetyRejectsPremineEnabledV1(t *testing.T) {
	clearFaucetPremineEnvV1(t)
	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_TESTNET_PREMINE_ENABLED", "1")

	if err := ValidateTestnetFaucetPremineSafetyV1(); err == nil {
		t.Fatalf("expected enabled testnet premine rejection")
	}
}

func TestValidateTestnetFaucetPremineSafetyRejectsMainAddressFallbackV1(t *testing.T) {
	clearFaucetPremineEnvV1(t)
	t.Setenv("QC_CHAIN_MODE", "testnet")
	t.Setenv("QC_MAIN_ADDRESS", "QC_TEST_MAIN_ADDRESS_SHOULD_NOT_BE_USED")

	if err := ValidateTestnetFaucetPremineSafetyV1(); err == nil {
		t.Fatalf("expected QC_MAIN_ADDRESS rejection in testnet mode")
	}
}

func TestValidateTestnetFaucetPremineSafetyRejectsPremineEnvV1(t *testing.T) {
	clearFaucetPremineEnvV1(t)
	t.Setenv("QC_NETWORK", "testnet")
	t.Setenv("QC_PREMINE_ADDRESS", "QC_TEST_PREMINE_ADDRESS_SHOULD_NOT_BE_USED")

	if err := ValidateTestnetFaucetPremineSafetyV1(); err == nil {
		t.Fatalf("expected QC_PREMINE_ADDRESS rejection in testnet mode")
	}
}

func TestValidateTestnetFaucetPremineSafetyRejectsNonZeroPreminePercentV1(t *testing.T) {
	clearFaucetPremineEnvV1(t)
	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_PREMINE_PERCENT", "5")

	if err := ValidateTestnetFaucetPremineSafetyV1(); err == nil {
		t.Fatalf("expected non-zero premine percent rejection in testnet mode")
	}
}

func TestValidateTestnetFaucetPremineSafetyIgnoresMainnetModeV1(t *testing.T) {
	clearFaucetPremineEnvV1(t)
	t.Setenv("QC_MAIN_ADDRESS", "MAINNET_ALLOWED_OUTSIDE_TESTNET")

	if err := ValidateTestnetFaucetPremineSafetyV1(); err != nil {
		t.Fatalf("mainnet/local env should not be rejected by testnet-only guard: %v", err)
	}
}
