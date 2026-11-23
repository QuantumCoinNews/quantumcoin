package blockchain

import (
	"fmt"
	"math/big"
)

// chainWork sums the expected hashes for already validated non-genesis blocks.
// PoW accepts hash < 2^(256-bits), so each block contributes exactly 2^bits.
// Claimed difficulty is never a substitute for canonical block validation.
func chainWork(blocks []*Block) (*big.Int, error) {
	if len(blocks) == 0 || blocks[0] == nil {
		return nil, ErrEmptyChain
	}
	work := new(big.Int)
	for i, block := range blocks[1:] {
		if block == nil || block.Difficulty < 1 || block.Difficulty > maxBits {
			return nil, fmt.Errorf("chain work: invalid block difficulty at %d", i+1)
		}
		work.Add(work, new(big.Int).Lsh(big.NewInt(1), uint(block.Difficulty)))
	}
	return work, nil
}

func greaterChainWork(candidate, local []*Block) (bool, error) {
	candidateWork, err := chainWork(candidate)
	if err != nil {
		return false, err
	}
	localWork, err := chainWork(local)
	if err != nil {
		return false, err
	}
	// Equal work keeps the current tip, preventing peer-driven oscillation.
	return candidateWork.Cmp(localWork) > 0, nil
}
