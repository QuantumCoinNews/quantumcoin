package node

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"

	"quantumcoin/blockchain"
	"quantumcoin/consensus"
)

// MineOnceContextV1 Fresh-network CPU mining hashes a detached canonical template. No chain or
// runtime lock is held while searching, so peer/Telegram blocks can advance tip.
func (r *Runtime) MineOnceContextV1(ctx context.Context, miner string, difficulty int) (*blockchain.Block, error) {
	ctx, cancel, err := r.miningContextV1(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	// Queue before reading the tip: a legacy one-shot may own both chain
	// and runtime read locks. A canceled queued caller need not wait for it.
	if err := r.lockMiningContextV1(ctx); err != nil {
		return nil, err
	}
	state := r.MiningTip()
	r.miningMu.Unlock()
	if state.GenesisDomain != blockchain.TestnetGenesisDomainV4 && state.GenesisDomain != blockchain.MainnetGenesisDomainV4 {
		return r.mineOnceContextV1(ctx, miner, difficulty, "")
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		template, header, err := r.PrepareExternalMining(miner)
		if err != nil {
			return nil, err
		}
		if template.Difficulty != difficulty {
			return nil, fmt.Errorf("mining difficulty differs from canonical template")
		}
		target := consensus.TargetFromDifficultyV1(template.Difficulty)
		var value big.Int
		for nonce := uint64(0); nonce <= blockchain.MaxConsensusNonceV1; nonce++ {
			if nonce%1024 == 0 {
				if err = ctx.Err(); err != nil {
					return nil, err
				}
				current := r.MiningTip()
				if current.GenesisHash != state.GenesisHash {
					return nil, ErrMiningNetworkMismatchV1
				}
				if current.Hash != fmt.Sprintf("%x", template.PrevHash) {
					break
				}
			}
			binary.BigEndian.PutUint64(header[len(header)-8:], nonce)
			hash := sha256.Sum256(header)
			if value.SetBytes(hash[:]).Cmp(target) < 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				block, commitErr := r.CommitExternalMining(template, nonce)
				if commitErr == nil {
					return block, nil
				}
				// An intervening accepted block is ordinary competition. Rebuild
				// from its hash, including same-height reorganizations.
				if r.MiningTip().Hash != fmt.Sprintf("%x", template.PrevHash) {
					break
				}
				return nil, commitErr
			}
		}
		// Re-read effective difficulty on each new chain tip; callers cannot
		// impose an old difficulty after a retarget.
		chain, cfg := r.Snapshot()
		if chain == nil || cfg == nil {
			return nil, ErrMiningBlockchainUnavailableV1
		}
		difficulty = chain.NextDifficultySnapshot(cfg.DefaultDifficultyBits)
	}
}
