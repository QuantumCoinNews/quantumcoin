package blockchain

import (
	"fmt"
	"io"
	"strings"
)

// MineBlockWithNFTAwardAndPersistSynchronizedV1 uses the existing local NFT
// approval and reward/fee engine. A successful file install precedes any live
// state mutation or advisory hook. Difficulty is derived under the same lock.
// The filename must come from trusted node configuration, never award data.
func (bc *Blockchain) MineBlockWithNFTAwardAndPersistSynchronizedV1(
	miner string,
	record NFTMintRecordV1,
	proof NFTAwardProofV1,
	filename string,
) (*Block, error) {
	if bc == nil {
		return nil, ErrChainNotInitialized
	}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return nil, fmt.Errorf("persistent NFT mining: chain filename is empty")
	}
	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	difficulty, err := bc.ExpectedNextDifficultyV1()
	if err != nil {
		return nil, fmt.Errorf("persistent NFT mining difficulty: %w", err)
	}
	return bc.mineNFTAwardWithCommitLockedV1(miner, difficulty, record, proof, filename)
}

// persistNFTMiningCandidateLockedV1 serializes a detached prospective chain.
// It does not append, clear the mempool or update live indexes/ownership.
// The caller already validated the candidate and owns the canonical lock.
// It shares the destination/OS lock and atomic installation primitive used
// by normal saves and worker commits, before publishing live NFT state.
func (bc *Blockchain) persistNFTMiningCandidateLockedV1(candidate *Block, filename string) error {
	if candidate == nil {
		return ErrNilBlock
	}
	if err := bc.validateBoundNetworkIdentityLockedV1(); err != nil {
		return err
	}
	prospective := &Blockchain{
		StoredNetworkIdentityV1: cloneNetworkIdentityV1(bc.networkIdentity),
		networkIdentity:         cloneNetworkIdentityV1(bc.networkIdentity),
		Blocks:                  cloneBlocksV1(bc.Blocks),
		TotalSupply:             bc.TotalSupply,
	}
	prospective.Blocks = append(prospective.Blocks, cloneBlockV1(candidate))
	prospective.UpdateUTXOSet()
	if err := persistChainFileV1(filename, func(writer io.Writer) error {
		return encodeDiskChainV1(writer, prospective)
	}, bc.networkIdentity); err != nil {
		return fmt.Errorf("persistent NFT mining: %w", err)
	}
	return nil
}
