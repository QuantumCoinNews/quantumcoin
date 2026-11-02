package config

import (
	"errors"
	"os"
	"strings"
)

// QCTestnetFaucetPremineSecurityVersionV1 documents the QuantumCoin component used by this module.
// QC_TESTNET_FAUCET_PREMINE_SECURITY_V1 prevents testnet from accidentally
// using real premine/faucet paths before explicit later approval.
const QCTestnetFaucetPremineSecurityVersionV1 = "QC_TESTNET_FAUCET_PREMINE_SECURITY_V1"

func faucetPremineEnvBoolV1(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func faucetPremineEnvSetV1(name string) bool {
	return strings.TrimSpace(os.Getenv(name)) != ""
}

func ValidateTestnetFaucetPremineSafetyV1() error {
	if !IsTestnetModeEnvV1() {
		return nil
	}

	if faucetPremineEnvBoolV1("QC_TESTNET_FAUCET_ENABLED") ||
		faucetPremineEnvBoolV1("QC_FAUCET_ENABLED") ||
		faucetPremineEnvBoolV1("QC_PUBLIC_FAUCET_ENABLED") {
		return errors.New("testnet faucet must remain disabled before explicit faucet approval")
	}

	for _, name := range []string{
		"QC_FAUCET_ADDRESS",
		"QC_FAUCET_PRIVATE_KEY",
		"QC_FAUCET_PRIVATE_KEY_HEX",
		"QC_TESTNET_FAUCET_ADDRESS",
		"QC_TESTNET_FAUCET_PRIVATE_KEY",
		"QC_TESTNET_FAUCET_PRIVATE_KEY_HEX",
	} {
		if faucetPremineEnvSetV1(name) {
			return errors.New("testnet faucet credential/address env must not be set before explicit faucet approval")
		}
	}

	if faucetPremineEnvBoolV1("QC_TESTNET_PREMINE_ENABLED") {
		return errors.New("testnet premine must remain disabled before explicit premine approval")
	}

	for _, name := range []string{
		"QC_MAIN_ADDRESS",
		"QC_PREMINE_ADDRESS",
		"QC_PREMINE_LIQUID_ADDRESS",
		"QC_PREMINE_LOCKED_ADDRESS",
	} {
		if faucetPremineEnvSetV1(name) {
			return errors.New("testnet premine/main address env must not be set before explicit premine approval")
		}
	}

	for _, name := range []string{
		"QC_PREMINE_PERCENT",
		"QC_PREMINE_LIQUID_PERCENT",
		"QC_PREMINE_LOCKED_PERCENT",
	} {
		v := strings.TrimSpace(os.Getenv(name))
		if v != "" && v != "0" {
			return errors.New("testnet premine percent env must be empty or zero before explicit premine approval")
		}
	}

	return nil
}
