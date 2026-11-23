package blockchain

import (
	"encoding/hex"
	"errors"
)

// CanonicalBlockIdentityV1 is detached from mutable block objects.
type CanonicalBlockIdentityV1 struct {
	Hash  string
	Miner string
}

// CanonicalBlockViewV1 describes selected heights from one canonical chain.
type CanonicalBlockViewV1 struct {
	TipHeight int
	Blocks    map[int]CanonicalBlockIdentityV1
}

// WithCanonicalBlockViewV1 excludes synchronized chain writers throughout
// visit, including a dependent ledger commit. visit must not call back into
// Blockchain methods: recursive read locks can deadlock behind a waiting writer.
// Only detached identities are exposed; modifying the view cannot change blocks.
func (bc *Blockchain) WithCanonicalBlockViewV1(heights []int, visit func(CanonicalBlockViewV1) error) error {
	if bc == nil {
		return ErrChainNotInitialized
	}
	if visit == nil {
		return errors.New("canonical block view requires a visitor")
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	view := CanonicalBlockViewV1{TipHeight: -1, Blocks: make(map[int]CanonicalBlockIdentityV1)}
	wanted := make(map[int]bool, len(heights))
	for _, height := range heights {
		wanted[height] = true
	}
	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}
		if block.Index > view.TipHeight {
			view.TipHeight = block.Index
		}
		if wanted[block.Index] {
			view.Blocks[block.Index] = CanonicalBlockIdentityV1{Hash: hex.EncodeToString(block.Hash), Miner: block.Miner}
		}
	}
	return visit(view)
}
