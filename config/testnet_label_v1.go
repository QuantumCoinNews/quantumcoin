package config

import "strings"

// QCTestnetLabelUIAPIVersionV1 documents the QuantumCoin component used by this module.
// QC_TESTNET_LABEL_UI_API_V1 exposes clear network/testnet label metadata for UI/API.
const QCTestnetLabelUIAPIVersionV1 = "QC_TESTNET_LABEL_UI_API_V1"

type NetworkLabelInfoV1 struct {
	Version     string `json:"version"`
	Network     string `json:"network"`
	Label       string `json:"label"`
	IsTestnet   bool   `json:"is_testnet"`
	Badge       string `json:"badge"`
	Warning     string `json:"warning,omitempty"`
	ChainFile   string `json:"chain_file,omitempty"`
	NetMagic    string `json:"net_magic,omitempty"`
	GenesisHash string `json:"genesis_hash,omitempty"`
}

func GetNetworkLabelInfoV1(chainFile string) NetworkLabelInfoV1 {
	profile, err := ResolveNetworkProfileV1(chainFile)
	if err != nil {
		return NetworkLabelForIdentityV1(chainFile, "", false)
	}
	network := string(profile.Profile)
	if profile.IsTestnet {
		network = QCNetworkTestnetV1
	}
	return NetworkLabelForIdentityV1(chainFile, network, profile.ExpectedChainFile == QCIsolatedLocalChainFileV1)
}

// NetworkLabelForIdentityV1 formats an already selected network. Public HTTP
// handlers pass the pinned chain identity rather than environment or path hints.
func NetworkLabelForIdentityV1(chainFile, network string, isolatedLocal bool) NetworkLabelInfoV1 {
	info := NetworkLabelInfoV1{
		Version: QCTestnetLabelUIAPIVersionV1, Network: network,
		ChainFile: chainFileBaseNameV1(chainFile),
	}
	switch network {
	case QCNetworkTestnetV1:
		info.Label, info.NetMagic, info.IsTestnet = "TESTNET", QCTestnetMagicV1, true
		info.Warning = "TESTNET MODE - not mainnet funds"
	case QCNetworkMainnetV1:
		info.Label, info.NetMagic = "MAINNET", QCMainnetMagicV1
	case string(QCNetworkProfileLocalReleaseV1):
		info.Label, info.NetMagic = "LOCAL RELEASE", "QC_LOCAL_RELEASE_V1"
		if isolatedLocal {
			info.Label, info.NetMagic = "ISOLATED LOCAL RELEASE", "QC_LOCAL_RELEASE_V2"
		}
	default:
		info.Network, info.Label = "unknown", "UNVERIFIED"
	}
	info.Badge = info.Label
	return info
}

func RequireVisibleTestnetLabelV1(chainFile string, visibleLabel string) error {
	info := GetNetworkLabelInfoV1(chainFile)
	if !info.IsTestnet {
		return nil
	}
	if !strings.Contains(strings.ToUpper(strings.TrimSpace(visibleLabel)), "TESTNET") {
		return ErrVisibleTestnetLabelRequiredV1
	}
	return nil
}
