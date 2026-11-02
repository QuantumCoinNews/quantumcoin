// Package miner contains QuantumCoin project code for this module.
package miner

import (
	"log"
	"time"

	"quantumcoin/blockchain"
)

// Compatibility hooks remain available to existing callers. The retired
// mining path never invokes them or represents an NFT award as successful.
var SaveNFTRewardHook func(address, nftType, txID string, at time.Time) error
var NotifyTelegramHook func(address, message string)

type NFTMiningReward struct {
	Address string
	Type    string
	Time    time.Time
	TxID    string
}

// GrantNFTBonus is a retired mining compatibility entry point.
// Monthly NFTs belong to the Telegram Game program. Neither legacy ENV flag
// can enable a mining award, chain mutation, database record or notification.
func GrantNFTBonus(address, nftType string, bc *blockchain.Blockchain) {
	log.Printf("[NFT_SAFE_BOUNDARY_V1] miner NFT reward skipped for %s (%s): mining awards are disabled; monthly Telegram Game program required", address, nftType)
}

func GenerateRandomNFTType() string {
	return "rare"
}
