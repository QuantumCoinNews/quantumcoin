package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
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
// This is the configured single writer's synchronous I/O boundary, not a
// cross-process lock or a claim of crash-atomic Rename on every platform.
func (bc *Blockchain) persistNFTMiningCandidateLockedV1(candidate *Block, filename string) error {
	if candidate == nil {
		return ErrNilBlock
	}
	prospective := &Blockchain{
		Blocks:      cloneBlocksV1(bc.Blocks),
		TotalSupply: bc.TotalSupply,
	}
	prospective.Blocks = append(prospective.Blocks, cloneBlockV1(candidate))
	prospective.UpdateUTXOSet()
	var encoded bytes.Buffer
	if err := gob.NewEncoder(&encoded).Encode(prospective); err != nil {
		return fmt.Errorf("persistent NFT mining encode: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(filename), ".qma-nft-commit-*")
	if err != nil {
		return fmt.Errorf("persistent NFT mining temporary file: %w", err)
	}
	temporary := file.Name()
	defer func() {
		_ = file.Close()
		_ = os.Remove(temporary)
	}()
	if _, err := file.Write(encoded.Bytes()); err != nil {
		return fmt.Errorf("persistent NFT mining write: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("persistent NFT mining flush: %w", err)
	}
	// Close the temporary handle before installing the completed file.
	if err := file.Close(); err != nil {
		return fmt.Errorf("persistent NFT mining close: %w", err)
	}
	if err := os.Rename(temporary, filename); err != nil {
		return fmt.Errorf("persistent NFT mining install: %w", err)
	}
	return nil
}
