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

	publicTestnet :=
		networkProfileEnvBoolV1("QC_PUBLIC_TESTNET") ||
			networkProfileEnvBoolV1("QC_TESTNET_PUBLIC_API")

	testnetRequested :=
		publicTestnet ||
			IsTestnetModeEnvV1() ||
			chainLooksTestnet

	mainnetRequested :=
		networkProfileEnvBoolV1("QC_MAINNET") ||
			networkProfileEnvContainsV1(
				"QC_NETWORK",
				"mainnet",
			) ||
			networkProfileEnvContainsV1(
				"QC_NETWORK_MODE",
				"mainnet",
			) ||
			networkProfileEnvContainsV1(
				"QC_CHAIN_MODE",
				"mainnet",
			) ||
			networkProfileEnvContainsV1(
				"QC_MODE",
				"mainnet",
			) ||
			strings.EqualFold(
				strings.TrimSpace(
					os.Getenv("QC_NET_MAGIC"),
				),
				QCMainnetMagicV1,
			)

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

	return networkProfileInfoV1(
		QCNetworkProfileLocalReleaseV1,
	), nil
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
			CanonicalDefaultDifficultyBits: 8,
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
			CanonicalDefaultDifficultyBits: 8,
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

func networkProfileEnvContainsV1(
	key string,
	needle string,
) bool {
	return strings.Contains(
		strings.ToLower(
			strings.TrimSpace(os.Getenv(key)),
		),
		strings.ToLower(needle),
	)
}
