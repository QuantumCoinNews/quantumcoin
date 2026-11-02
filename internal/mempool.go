// internal/mempool.go

// Package internal contains QuantumCoin project code for this module.
package internal

import (
	"bytes"
	"sync"

	"quantumcoin/blockchain"
)

// Mempool is a compatibility adapter over the single canonical
// blockchain.Blockchain-owned pending transaction pool.
//
// It never owns an independent transaction collection. Admission is disabled
// until a canonical Blockchain is explicitly bound.
type Mempool struct {
	mu        sync.RWMutex
	canonical *blockchain.Blockchain
	capacity  int
}

// NewMempool creates an unbound, fail-closed compatibility adapter.
func NewMempool() *Mempool {
	return &Mempool{}
}

// NewCanonicalMempool creates an adapter bound to the supplied canonical
// Blockchain. A nil Blockchain leaves the adapter fail-closed.
func NewCanonicalMempool(
	canonical *blockchain.Blockchain,
) *Mempool {
	mp := NewMempool()
	mp.BindCanonicalBlockchain(canonical)
	return mp
}

// BindCanonicalBlockchain binds the adapter to one canonical Blockchain.
// Rebinding to another Blockchain is rejected so authority cannot silently
// switch after initialization.
func (mp *Mempool) BindCanonicalBlockchain(
	canonical *blockchain.Blockchain,
) bool {
	if mp == nil || canonical == nil {
		return false
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.canonical != nil && mp.canonical != canonical {
		return false
	}

	mp.canonical = canonical
	return true
}

// SetCapacity applies an adapter-local admission ceiling. The canonical
// Blockchain remains the owner of accepted transactions.
func (mp *Mempool) SetCapacity(n int) {
	if mp == nil {
		return
	}

	mp.mu.Lock()
	mp.capacity = n
	mp.mu.Unlock()
}

// Len returns the canonical pending transaction count.
func (mp *Mempool) Len() int {
	canonical := mp.canonicalSnapshot()
	if canonical == nil {
		return 0
	}

	return canonical.PendingTransactionCountSnapshot()
}

// Has reports whether the canonical pending pool contains txID.
func (mp *Mempool) Has(txID []byte) bool {
	if len(txID) == 0 {
		return false
	}

	canonical := mp.canonicalSnapshot()
	if canonical == nil {
		return false
	}

	for _, tx := range canonical.PendingTransactionsSnapshot() {
		if tx != nil && bytes.Equal(tx.ID, txID) {
			return true
		}
	}

	return false
}

// Add delegates admission to Blockchain.AddTransaction, the sole authority for
// transaction basics, signatures, UTXO ownership, input/output totals, fees,
// coinbase maturity and pending-outpoint double-spend protection.
func (mp *Mempool) Add(tx *blockchain.Transaction) bool {
	if mp == nil || tx == nil {
		return false
	}

	// Coinbase creation belongs only to canonical block construction.
	if tx.IsCoinbase() {
		return false
	}

	if len(tx.ID) == 0 {
		return false
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	canonical := mp.canonical
	if canonical == nil {
		return false
	}

	if mp.capacity > 0 &&
		canonical.PendingTransactionCountSnapshot() >= mp.capacity {
		return false
	}

	return canonical.AddTransaction(tx) == nil
}

// GetTransactions returns detached canonical pending transactions.
func (mp *Mempool) GetTransactions() []*blockchain.Transaction {
	canonical := mp.canonicalSnapshot()
	if canonical == nil {
		return []*blockchain.Transaction{}
	}

	return canonical.PendingTransactionsSnapshot()
}

// PopBatch is retained for source compatibility but cannot remove transactions
// from the canonical pool. Mining and canonical block acceptance own cleanup.
func (mp *Mempool) PopBatch(n int) []*blockchain.Transaction {
	_ = n
	return nil
}

// RemoveTx is retained for source compatibility but cannot mutate canonical
// pending state.
func (mp *Mempool) RemoveTx(txID []byte) bool {
	_ = txID
	return false
}

// Clear is retained for source compatibility but cannot mutate canonical
// pending state.
func (mp *Mempool) Clear() {
}

// canonicalSnapshot reads the bound authority without exposing adapter state.
func (mp *Mempool) canonicalSnapshot() *blockchain.Blockchain {
	if mp == nil {
		return nil
	}

	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return mp.canonical
}
