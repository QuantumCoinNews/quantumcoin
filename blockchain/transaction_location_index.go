package blockchain

import "encoding/hex"

// TransactionLocation is a point-in-time lookup result for one canonical
// transaction identifier.
type TransactionLocation struct {
	InBlock         bool
	InMempool       bool
	ConfirmedHeight int
}

func unknownTransactionLocation() TransactionLocation {
	return TransactionLocation{
		ConfirmedHeight: -1,
	}
}

func transactionLocationIndexKey(txID []byte) string {
	if len(txID) == 0 {
		return ""
	}

	return hex.EncodeToString(txID)
}

// rebuildTransactionLocationIndexesLocked rebuilds ephemeral lookup indexes
// from authoritative Blocks and pendingTxs state.
//
// The caller must own stateMu exclusively, or the Blockchain must not yet
// have been published to other goroutines.
func (bc *Blockchain) rebuildTransactionLocationIndexesLocked() {
	if bc == nil {
		return
	}

	confirmed := make(map[string]int)

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}

		for _, tx := range block.Transactions {
			if tx == nil {
				continue
			}

			key := transactionLocationIndexKey(tx.ID)
			if key == "" {
				continue
			}

			// Preserve the old forward full-chain scan contract:
			// first confirmed occurrence wins.
			if _, exists := confirmed[key]; !exists {
				confirmed[key] = block.Index
			}
		}
	}

	pending := make(map[string]struct{})

	for _, tx := range bc.pendingTxs {
		if tx == nil {
			continue
		}

		key := transactionLocationIndexKey(tx.ID)
		if key == "" {
			continue
		}

		pending[key] = struct{}{}
	}

	bc.confirmedTxIndex = confirmed
	bc.pendingTxIndex = pending
}

func (bc *Blockchain) ensureTransactionLocationIndexesLocked() {
	if bc == nil {
		return
	}

	if bc.confirmedTxIndex != nil &&
		bc.pendingTxIndex != nil {
		return
	}

	bc.rebuildTransactionLocationIndexesLocked()
}

// indexConfirmedBlockTransactionsLocked updates the confirmed index after an
// authoritative block append.
//
// The caller must own stateMu exclusively.
func (bc *Blockchain) indexConfirmedBlockTransactionsLocked(
	block *Block,
) {
	if bc == nil || block == nil {
		return
	}

	bc.ensureTransactionLocationIndexesLocked()

	for _, tx := range block.Transactions {
		if tx == nil {
			continue
		}

		key := transactionLocationIndexKey(tx.ID)
		if key == "" {
			continue
		}

		if _, exists := bc.confirmedTxIndex[key]; !exists {
			bc.confirmedTxIndex[key] = block.Index
		}
	}
}

// indexPendingTransactionLocked updates the pending index after canonical
// mempool admission succeeds.
//
// The caller must own stateMu exclusively.
func (bc *Blockchain) indexPendingTransactionLocked(
	tx *Transaction,
) {
	if bc == nil || tx == nil {
		return
	}

	bc.ensureTransactionLocationIndexesLocked()

	key := transactionLocationIndexKey(tx.ID)
	if key == "" {
		return
	}

	bc.pendingTxIndex[key] = struct{}{}
}

// clearPendingTransactionIndexLocked mirrors an authoritative pendingTxs
// clear.
//
// The caller must own stateMu exclusively.
func (bc *Blockchain) clearPendingTransactionIndexLocked() {
	if bc == nil {
		return
	}

	bc.pendingTxIndex = make(map[string]struct{})
}

func (bc *Blockchain) transactionLocationLocked(
	txIDHex string,
) TransactionLocation {
	result := unknownTransactionLocation()

	if bc == nil || txIDHex == "" {
		return result
	}

	if height, exists := bc.confirmedTxIndex[txIDHex]; exists {
		result.InBlock = true
		result.ConfirmedHeight = height
	}

	if _, exists := bc.pendingTxIndex[txIDHex]; exists {
		result.InMempool = true
	}

	return result
}

// TransactionLocationSnapshot returns O(1) transaction location information
// through the canonical Blockchain synchronization boundary.
//
// Indexes are unexported and ephemeral. Deserialization/reorg rebuild them
// from authoritative state.
func (bc *Blockchain) TransactionLocationSnapshot(
	txIDHex string,
) TransactionLocation {
	if bc == nil || txIDHex == "" {
		return unknownTransactionLocation()
	}

	bc.stateMu.RLock()

	if bc.confirmedTxIndex != nil &&
		bc.pendingTxIndex != nil {
		result := bc.transactionLocationLocked(txIDHex)
		bc.stateMu.RUnlock()

		return result
	}

	bc.stateMu.RUnlock()

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	bc.ensureTransactionLocationIndexesLocked()

	return bc.transactionLocationLocked(txIDHex)
}
