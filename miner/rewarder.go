package miner

import "log"

// GrantNFTReward is retained as an inert mining compatibility entry point.
// NFT awards must be confirmed by the separate monthly Telegram Game flow.
func GrantNFTReward(address string) {
	log.Printf("[NFT_SAFE_BOUNDARY_V1] miner NFT reward skipped for %s: mining awards are disabled; monthly Telegram Game program required", address)
}
