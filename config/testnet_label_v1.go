package config

import "strings"

// QCTestnetLabelUIAPIVersionV1 documents the QuantumCoin component used by this module.
// QC_TESTNET_LABEL_UI_API_V1 exposes clear network/testnet label metadata for UI/API.
const QCTestnetLabelUIAPIVersionV1 = "QC_TESTNET_LABEL_UI_API_V1"

type NetworkLabelInfoV1 struct {
	Version   string `json:"version"`
	Network   string `json:"network"`
	Label     string `json:"label"`
	IsTestnet bool   `json:"is_testnet"`
	Badge     string `json:"badge"`
	Warning   string `json:"warning,omitempty"`
	ChainFile string `json:"chain_file,omitempty"`
	NetMagic  string `json:"net_magic,omitempty"`
}

func GetNetworkLabelInfoV1(chainFile string) NetworkLabelInfoV1 {
	cf := strings.TrimSpace(chainFile)
	base := chainFileBaseNameV1(cf)

	isTestnet := IsTestnetModeEnvV1() || strings.Contains(base, "testnet")

	if isTestnet {
		return NetworkLabelInfoV1{
			Version:   QCTestnetLabelUIAPIVersionV1,
			Network:   QCNetworkTestnetV1,
			Label:     "TESTNET",
			IsTestnet: true,
			Badge:     "TESTNET",
			Warning:   "TESTNET MODE - not mainnet funds",
			ChainFile: cf,
			NetMagic:  QCTestnetMagicV1,
		}
	}

	return NetworkLabelInfoV1{
		Version:   QCTestnetLabelUIAPIVersionV1,
		Network:   QCNetworkMainnetV1,
		Label:     "MAINNET",
		IsTestnet: false,
		Badge:     "MAINNET",
		ChainFile: cf,
		NetMagic:  QCMainnetMagicV1,
	}
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
