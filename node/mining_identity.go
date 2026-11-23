package node

import (
	"quantumcoin/blockchain"
	"quantumcoin/config"
)

// TrustedMiningIdentity derives the same genesis as bootstrap without reading
// a chain file, creating a wallet or starting any node service.
func TrustedMiningIdentity(cfg *config.Config) (blockchain.MiningTip, error) {
	chain, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		return blockchain.MiningTip{}, err
	}
	return chain.MiningTipSnapshot(), nil
}
