package config

import (
	"errors"
	"os"
	"strings"
)

// QCChainFileSeparationVersionV1 documents the QuantumCoin component used by this module.
// QC_CHAIN_FILE_SEPARATION_V1 prevents mainnet and testnet from sharing the same chain file.
const QCChainFileSeparationVersionV1 = "QC_CHAIN_FILE_SEPARATION_V1"

func chainFileEnvBoolV1(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func chainFileEnvContainsV1(name string, needle string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(os.Getenv(name))), strings.ToLower(needle))
}

func IsTestnetModeEnvV1() bool {
	if chainFileEnvBoolV1("QC_TESTNET") ||
		chainFileEnvBoolV1("QC_PUBLIC_TESTNET") ||
		chainFileEnvBoolV1("QC_API_PUBLIC_TESTNET") ||
		chainFileEnvBoolV1("QC_TESTNET_PUBLIC_API") {
		return true
	}

	return chainFileEnvContainsV1("QC_NET_MAGIC", "testnet") ||
		chainFileEnvContainsV1("QC_NETWORK", "testnet") ||
		chainFileEnvContainsV1("QC_NETWORK_MODE", "testnet") ||
		chainFileEnvContainsV1("QC_CHAIN_MODE", "testnet") ||
		chainFileEnvContainsV1("QC_MODE", "testnet")
}

func chainFileBaseNameV1(path string) string {
	s := strings.TrimSpace(path)
	s = strings.ReplaceAll(s, "\\", "/")
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
}

func ValidateChainFilePathForModeV1(chainFile string) error {
	cf := strings.TrimSpace(chainFile)
	if cf == "" {
		return errors.New("chain file is required")
	}

	base := chainFileBaseNameV1(cf)
	if base == "" {
		return errors.New("chain file base name is required")
	}

	isMainFile := base == QCMainnetChainFileV1
	isTestnetFile := strings.Contains(base, "testnet")
	testnetMode := IsTestnetModeEnvV1()

	if testnetMode {
		if isMainFile {
			return errors.New("testnet mode must not use mainnet chain_data.dat")
		}
		if !isTestnetFile {
			return errors.New("testnet mode requires a clearly labeled testnet chain file")
		}
		return nil
	}

	if isTestnetFile {
		return errors.New("testnet chain file requires explicit testnet mode")
	}

	return nil
}

func ValidateChainFileSeparationV1(c *Config) error {
	if c == nil {
		return errors.New("config is nil")
	}
	return ValidateChainFilePathForModeV1(c.ChainFile)
}
