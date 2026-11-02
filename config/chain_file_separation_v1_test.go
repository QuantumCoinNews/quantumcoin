package config

import "testing"

func clearChainFileModeEnvV1(t *testing.T) {
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
		"QC_CHAIN_FILE",
	} {
		t.Setenv(name, "")
	}
}

func TestValidateChainFileSeparationAllowsMainnetDefaultV1(t *testing.T) {
	clearChainFileModeEnvV1(t)

	c := Default()
	c.ChainFile = QCMainnetChainFileV1

	if err := ValidateChainFileSeparationV1(c); err != nil {
		t.Fatalf("mainnet default chain file should be valid: %v", err)
	}
}

func TestValidateChainFileSeparationRejectsTestnetModeMainFileV1(t *testing.T) {
	clearChainFileModeEnvV1(t)
	t.Setenv("QC_TESTNET", "1")

	c := Default()
	c.ChainFile = QCMainnetChainFileV1

	if err := ValidateChainFileSeparationV1(c); err == nil {
		t.Fatalf("expected testnet mode to reject mainnet chain file")
	}
}

func TestValidateChainFileSeparationAllowsTestnetModeTestnetFileV1(t *testing.T) {
	clearChainFileModeEnvV1(t)
	t.Setenv("QC_TESTNET", "1")

	c := Default()
	c.ChainFile = QCTestnetChainFileV1

	if err := ValidateChainFileSeparationV1(c); err != nil {
		t.Fatalf("testnet chain file should be valid in testnet mode: %v", err)
	}
}

func TestValidateChainFileSeparationRejectsTestnetFileWithoutModeV1(t *testing.T) {
	clearChainFileModeEnvV1(t)

	c := Default()
	c.ChainFile = QCTestnetChainFileV1

	if err := ValidateChainFileSeparationV1(c); err == nil {
		t.Fatalf("expected testnet chain file to require explicit testnet mode")
	}
}

func TestValidateChainFileSeparationRejectsGenericFileInTestnetModeV1(t *testing.T) {
	clearChainFileModeEnvV1(t)
	t.Setenv("QC_NET_MAGIC", QCTestnetMagicV1)

	c := Default()
	c.ChainFile = "chain_data_custom.dat"

	if err := ValidateChainFileSeparationV1(c); err == nil {
		t.Fatalf("expected generic chain file to be rejected in testnet mode")
	}
}

func TestValidateChainFileSeparationAcceptsNestedTestnetPathV1(t *testing.T) {
	clearChainFileModeEnvV1(t)
	t.Setenv("QC_CHAIN_MODE", "testnet")

	c := Default()
	c.ChainFile = "data/testnet/chain_data_testnet.dat"

	if err := ValidateChainFileSeparationV1(c); err != nil {
		t.Fatalf("nested testnet path should be valid in testnet mode: %v", err)
	}
}
