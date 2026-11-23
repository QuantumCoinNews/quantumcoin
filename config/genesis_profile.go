package config

import "fmt"

const NetworkGenesisProfileV2 = "network-v2"
const NetworkGenesisProfileV3 = "network-v3"
const NetworkGenesisProfileV4 = "network-v4"

func ValidateGenesisProfileV1(value string, network NetworkProfileV1) error {
	switch value {
	case "":
		return nil
	case NetworkGenesisProfileV2, NetworkGenesisProfileV3, NetworkGenesisProfileV4:
		switch network {
		case QCNetworkProfileTestnetLocalV1, QCNetworkProfileTestnetPublicV1, QCNetworkProfileMainnetV1:
			return nil
		}
	}
	return fmt.Errorf("genesis_profile must be empty, network-v2, network-v3 or network-v4 on a testnet/mainnet profile")
}
