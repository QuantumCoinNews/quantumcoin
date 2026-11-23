package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type NetworkProfileV1 string

const (
	QCNetworkProfileLocalReleaseV1  NetworkProfileV1 = "local-release"
	QCNetworkProfileTestnetLocalV1  NetworkProfileV1 = "testnet-local"
	QCNetworkProfileTestnetPublicV1 NetworkProfileV1 = "testnet-public"
	QCNetworkProfileMainnetV1       NetworkProfileV1 = "mainnet"
)

const (
	QCTestnetDefaultDifficultyBitsV1 = 8
	QCIsolatedLocalChainFileV1       = "chain_data_local.dat"
	QCIsolatedLocalHTTPPortV1        = ":28082"
	QCIsolatedLocalP2PPortV1         = ":23001"
)

type NetworkProfileInfoV1 struct {
	Profile                        NetworkProfileV1 `json:"profile"`
	IsTestnet                      bool             `json:"is_testnet"`
	IsPublicTestnet                bool             `json:"is_public_testnet"`
	IsExplicitMainnet              bool             `json:"is_explicit_mainnet"`
	ExpectedChainFile              string           `json:"expected_chain_file"`
	ExpectedHTTPPort               string           `json:"expected_http_port"`
	ExpectedP2PPort                string           `json:"expected_p2p_port"`
	CanonicalDefaultDifficultyBits int              `json:"canonical_default_difficulty_bits"`
	AllowsFixedDifficultyOverride  bool             `json:"allows_fixed_difficulty_override"`
}

// ResolveNetworkProfileV1 classifies runtime intent without treating
// an unspecified local process as an activated mainnet process.
func ResolveNetworkProfileV1(
	chainFile string,
) (NetworkProfileInfoV1, error) {
	base := strings.ToLower(
		filepath.Base(strings.TrimSpace(chainFile)),
	)

	chainLooksTestnet := strings.Contains(base, "testnet")

	publicTestnet := networkProfileEnvBoolV1("QC_PUBLIC_TESTNET") ||
		networkProfileEnvBoolV1("QC_API_PUBLIC_TESTNET") ||
		networkProfileEnvBoolV1("QC_TESTNET_PUBLIC_API")
	testnetRequested := publicTestnet || networkProfileEnvBoolV1("QC_TESTNET") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("QC_NET_MAGIC")), QCTestnetMagicV1) || chainLooksTestnet
	mainnetRequested := networkProfileEnvBoolV1("QC_MAINNET") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("QC_NET_MAGIC")), QCMainnetMagicV1)
	localRequested := false
	for _, name := range networkSelectorKeysV1 {
		switch networkSelectorV1(name) {
		case "":
		case "local", "local-release":
			localRequested = true
		case "testnet", "testnet-local":
			testnetRequested = true
		case "testnet-public":
			testnetRequested, publicTestnet = true, true
		case "mainnet":
			mainnetRequested = true
		default:
			return NetworkProfileInfoV1{}, fmt.Errorf("%s contains an unknown network profile", name)
		}
	}
	for _, name := range []string{"QC_MAINNET", "QC_TESTNET", "QC_PUBLIC_TESTNET", "QC_API_PUBLIC_TESTNET", "QC_TESTNET_PUBLIC_API"} {
		switch networkSelectorV1(name) {
		case "", "1", "true", "yes", "on", "0", "false", "no", "off":
		default:
			return NetworkProfileInfoV1{}, fmt.Errorf("%s requires a boolean value", name)
		}
	}
	if localRequested && (testnetRequested || mainnetRequested) {
		return NetworkProfileInfoV1{}, fmt.Errorf("conflicting local and public network runtime signals")
	}

	if testnetRequested && mainnetRequested {
		return NetworkProfileInfoV1{}, fmt.Errorf(
			"conflicting testnet and mainnet runtime signals",
		)
	}

	if testnetRequested {
		if publicTestnet {
			return networkProfileInfoV1(
				QCNetworkProfileTestnetPublicV1,
			), nil
		}

		return networkProfileInfoV1(
			QCNetworkProfileTestnetLocalV1,
		), nil
	}

	if mainnetRequested {
		return networkProfileInfoV1(
			QCNetworkProfileMainnetV1,
		), nil
	}

	profile := networkProfileInfoV1(QCNetworkProfileLocalReleaseV1)
	if base == QCIsolatedLocalChainFileV1 {
		profile.ExpectedChainFile = QCIsolatedLocalChainFileV1
		profile.ExpectedHTTPPort = QCIsolatedLocalHTTPPortV1
		profile.ExpectedP2PPort = QCIsolatedLocalP2PPortV1
	}
	return profile, nil
}

func networkProfileInfoV1(
	profile NetworkProfileV1,
) NetworkProfileInfoV1 {
	switch profile {
	case QCNetworkProfileTestnetLocalV1:
		return NetworkProfileInfoV1{
			Profile:                        profile,
			IsTestnet:                      true,
			ExpectedChainFile:              QCTestnetChainFileV1,
			ExpectedHTTPPort:               ":18082",
			ExpectedP2PPort:                ":13001",
			CanonicalDefaultDifficultyBits: QCTestnetDefaultDifficultyBitsV1,
			AllowsFixedDifficultyOverride:  false,
		}

	case QCNetworkProfileTestnetPublicV1:
		return NetworkProfileInfoV1{
			Profile:                        profile,
			IsTestnet:                      true,
			IsPublicTestnet:                true,
			ExpectedChainFile:              QCTestnetChainFileV1,
			ExpectedHTTPPort:               ":18082",
			ExpectedP2PPort:                ":13001",
			CanonicalDefaultDifficultyBits: QCTestnetDefaultDifficultyBitsV1,
			AllowsFixedDifficultyOverride:  false,
		}

	case QCNetworkProfileMainnetV1:
		return NetworkProfileInfoV1{
			Profile:                        profile,
			IsExplicitMainnet:              true,
			ExpectedChainFile:              QCMainnetChainFileV1,
			ExpectedHTTPPort:               ":8082",
			ExpectedP2PPort:                ":3001",
			CanonicalDefaultDifficultyBits: 16,
			AllowsFixedDifficultyOverride:  false,
		}

	default:
		return NetworkProfileInfoV1{
			Profile:                        QCNetworkProfileLocalReleaseV1,
			ExpectedChainFile:              QCMainnetChainFileV1,
			ExpectedHTTPPort:               ":8082",
			ExpectedP2PPort:                ":3001",
			CanonicalDefaultDifficultyBits: 16,
			AllowsFixedDifficultyOverride:  true,
		}
	}
}

func networkProfileEnvBoolV1(key string) bool {
	switch strings.ToLower(
		strings.TrimSpace(os.Getenv(key)),
	) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

var networkSelectorKeysV1 = [...]string{"QC_NETWORK", "QC_NETWORK_MODE", "QC_CHAIN_MODE", "QC_MODE"}

func networkSelectorV1(key string) string {
	return strings.ToLower(strings.TrimSpace(os.Getenv(key)))
}
