package blockchain

import (
	"fmt"
	"os"
)

// CommitValidatedBlockAndPersist owns the local-miner block commit boundary.
// Validation, chain mutation, UTXO rebuild and persistence are serialized
// under the Blockchain state write lock.
//
// The committed block is deep-copied before entering live state so callers
// cannot mutate the canonical chain through a retained pointer.
func (bc *Blockchain) CommitValidatedBlockAndPersist(
	block *Block,
	filename string,
) error {
	if bc == nil {
		return fmt.Errorf("nil blockchain")
	}

	if block == nil {
		return ErrNilBlock
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	candidate := cloneBlockV1(block)

	if candidate == nil {
		return ErrNilBlock
	}

	// This validation path is intentionally executed while holding the
	// canonical write lock so the validated tip/UTXO view cannot change
	// before the block is committed.
	if err := bc.ValidatePeerBlockStrictV2(
		candidate,
	); err != nil {
		return fmt.Errorf(
			"local miner block validation: %w",
			err,
		)
	}

	oldLength := len(bc.Blocks)

	bc.Blocks = append(
		bc.Blocks,
		candidate,
	)

	// UpdateUTXOSet is currently the package-internal rebuild primitive.
	// The caller already owns stateMu exclusively here.
	bc.UpdateUTXOSet()

	tmp := filename + ".tmp"

	if err := os.WriteFile(
		tmp,
		SerializeBlockchain(bc),
		0o600,
	); err != nil {
		bc.Blocks = bc.Blocks[:oldLength]
		bc.UpdateUTXOSet()

		return fmt.Errorf(
			"persist committed block: %w",
			err,
		)
	}

	if err := os.Rename(
		tmp,
		filename,
	); err != nil {
		_ = os.Remove(tmp)

		bc.Blocks = bc.Blocks[:oldLength]
		bc.UpdateUTXOSet()

		return fmt.Errorf(
			"install committed blockchain file: %w",
			err,
		)
	}

	bc.indexConfirmedBlockTransactionsLocked(candidate)

	return nil
}
