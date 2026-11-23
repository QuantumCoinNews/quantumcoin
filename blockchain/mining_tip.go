package blockchain

import (
	"bytes"
	"encoding/hex"
)

// MiningTip is a small, coherent identity snapshot. It does not clone blocks,
// scan balances or alter consensus; equal-height reorganizations change Hash.
type MiningTip struct {
	Version       string `json:"version"`
	Height        int    `json:"height"`
	Hash          string `json:"hash"`
	GenesisHash   string `json:"genesis_hash"`
	Network       string `json:"network"`
	GenesisDomain string `json:"genesis_domain"`
}

// MiningResultSnapshot A height/hash lookup reconciles a lost mining response using canonical disk-
// replayed chain state, rather than an ephemeral job or a second submission.
func (bc *Blockchain) MiningResultSnapshot(height int, hash []byte, miner string) bool {
	if bc == nil || height < 1 || len(hash) != 32 {
		return false
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	if height >= len(bc.Blocks) {
		return false
	}
	block := bc.Blocks[height]
	return block != nil && block.Index == height && block.Miner == miner && bytes.Equal(block.Hash, hash)
}

func (bc *Blockchain) MiningTipSnapshot() MiningTip {
	result := MiningTip{Version: "QC_MINING_TIP_V1", Height: -1}
	if bc == nil {
		return result
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil || bc.Blocks[len(bc.Blocks)-1] == nil {
		return result
	}
	tip := bc.Blocks[len(bc.Blocks)-1]
	result.Height = tip.Index
	result.Hash = hex.EncodeToString(tip.Hash)
	result.GenesisHash = hex.EncodeToString(bc.Blocks[0].Hash)
	if bc.networkIdentity != nil {
		result.Network = bc.networkIdentity.Network
		result.GenesisDomain = bc.networkIdentity.GenesisDomain
	}
	return result
}
