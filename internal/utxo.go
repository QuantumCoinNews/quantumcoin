// internal/utxo.go

// Package internal contains QuantumCoin project code for this module.
package internal

import (
	"sync"

	"quantumcoin/blockchain"
)

// UTXOSet is a read-only compatibility adapter over the single canonical
// blockchain.Blockchain-owned UTXO state.
//
// It never owns or exports an independently mutable UTXO map. Spend validity,
// duplicate-spend checks, ownership, value conservation, fees and coinbase
// maturity remain exclusively under blockchain.Blockchain consensus rules.
type UTXOSet struct {
	mu        sync.RWMutex
	canonical *blockchain.Blockchain
}

// NewUTXOSet creates an unbound, fail-closed compatibility adapter.
func NewUTXOSet() *UTXOSet {
	return &UTXOSet{}
}

// NewCanonicalUTXOSet creates an adapter bound to canonical. A nil Blockchain
// leaves the adapter fail-closed.
func NewCanonicalUTXOSet(canonical *blockchain.Blockchain) *UTXOSet {
	set := NewUTXOSet()
	set.BindCanonicalBlockchain(canonical)
	return set
}

// BindCanonicalBlockchain binds this adapter to exactly one canonical chain.
// Idempotent rebinding to the same pointer is allowed; authority switching is
// rejected.
func (set *UTXOSet) BindCanonicalBlockchain(
	canonical *blockchain.Blockchain,
) bool {
	if set == nil || canonical == nil {
		return false
	}

	set.mu.Lock()
	defer set.mu.Unlock()

	if set.canonical != nil && set.canonical != canonical {
		return false
	}

	set.canonical = canonical
	return true
}

// Reindex preserves the legacy API while delegating the rebuild to the
// canonical synchronized state owner. It cannot create a second UTXO map.
func (set *UTXOSet) Reindex(chain *blockchain.Blockchain) {
	if !set.BindCanonicalBlockchain(chain) {
		return
	}

	canonical := set.canonicalSnapshot()
	if canonical == nil {
		return
	}

	canonical.UpdateUTXOSetSynchronized()
}

// Snapshot returns a detached copy of canonical UTXO state. Mutating the
// returned map, slices or output byte fields cannot mutate live chain state.
func (set *UTXOSet) Snapshot() map[string][]blockchain.TransactionOutput {
	canonical := set.canonicalSnapshot()
	if canonical == nil {
		return map[string][]blockchain.TransactionOutput{}
	}

	snapshot := canonical.StateSnapshot()
	if snapshot.UTXO == nil {
		return map[string][]blockchain.TransactionOutput{}
	}

	return snapshot.UTXO
}

// FindSpendableOutputs delegates selection to the canonical synchronized
// boundary, including canonical coinbase-maturity filtering.
func (set *UTXOSet) FindSpendableOutputs(
	pubKeyHash []byte,
	amount int,
) (map[string][]int, int) {
	if len(pubKeyHash) == 0 || amount <= 0 {
		return map[string][]int{}, 0
	}

	canonical := set.canonicalSnapshot()
	if canonical == nil {
		return map[string][]int{}, 0
	}

	spendable, accumulated := canonical.FindSpendableOutputsSnapshot(
		pubKeyHash,
		amount,
	)
	if spendable == nil {
		return map[string][]int{}, accumulated
	}

	return spendable, accumulated
}

// FindUTXOs returns detached, mature outputs through one coherent canonical
// read boundary.
func (set *UTXOSet) FindUTXOs(
	pubKeyHash []byte,
) []blockchain.TransactionOutput {
	if len(pubKeyHash) == 0 {
		return []blockchain.TransactionOutput{}
	}

	canonical := set.canonicalSnapshot()
	if canonical == nil {
		return []blockchain.TransactionOutput{}
	}

	return canonical.FindUTXOsSnapshot(pubKeyHash)
}

func (set *UTXOSet) canonicalSnapshot() *blockchain.Blockchain {
	if set == nil {
		return nil
	}

	set.mu.RLock()
	defer set.mu.RUnlock()

	return set.canonical
}
