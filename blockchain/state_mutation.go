package blockchain

import (
	"fmt"
	"io"
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
	if err := bc.validateBoundNetworkIdentityLockedV1(); err != nil {
		return err
	}

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

	if err := persistChainFileV1(filename, func(writer io.Writer) error {
		return encodeDiskChainV1(writer, bc)
	}, bc.networkIdentity); err != nil {
		bc.Blocks = bc.Blocks[:oldLength]
		bc.UpdateUTXOSet()

		return fmt.Errorf(
			"persist committed block: %w",
			err,
		)
	}

	bc.indexConfirmedBlockTransactionsLocked(candidate)
	// Work may contain mempool transactions. Remove confirmed entries while
	// retaining transactions admitted after this particular template was issued.
	confirmed := make(map[string]bool)
	for _, tx := range candidate.Transactions {
		confirmed[transactionLocationIndexKey(tx.ID)] = true
	}
	remaining := make([]*Transaction, 0, len(bc.pendingTxs))
	for _, tx := range bc.pendingTxs {
		if !confirmed[transactionLocationIndexKey(tx.ID)] {
			remaining = append(remaining, tx)
		}
	}
	bc.pendingTxs = remaining
	bc.rebuildTransactionLocationIndexesLocked()

	return nil
}
