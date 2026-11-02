package blockchain

import (
	"encoding/hex"

	qconsensus "quantumcoin/consensus"
)

// HeightSnapshot returns the canonical chain height through the read boundary.
func (bc *Blockchain) HeightSnapshot() int {
	if bc == nil {
		return -1
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return bc.GetHeight()
}

// TotalMintedSnapshot protects the complete historical minted-supply scan.
func (bc *Blockchain) TotalMintedSnapshot() int {
	if bc == nil {
		return 0
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return bc.TotalMinted()
}

// NextDifficultySnapshot calculates the next difficulty from one coherent
// chain view.
func (bc *Blockchain) NextDifficultySnapshot(
	defaultBits int,
) int {
	if bc == nil {
		return defaultBits
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return bc.NextDifficulty(defaultBits)
}

// ProtocolAtHeightSnapshotV1 protects activation-policy reads performed
// outside Blockchain-owned mutation boundaries.
func (bc *Blockchain) ProtocolAtHeightSnapshotV1(
	height int,
) qconsensus.ProtocolVersion {
	if bc == nil {
		return qconsensus.ProtocolLegacyV1
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return bc.ProtocolAtHeightV1(height)
}

// CalculateBlockFeesSnapshot uses the writer boundary because legacy UTXO
// validation may lazily initialize Blockchain state.
func (bc *Blockchain) CalculateBlockFeesSnapshot(
	transactions []*Transaction,
) (int, error) {
	if bc == nil {
		return 0, ErrNilBlockchain
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return bc.CalculateBlockFeesV1(transactions)
}

// FindSpendableOutputsSnapshot protects transaction construction from
// concurrent UTXO mutation.
func (bc *Blockchain) FindSpendableOutputsSnapshot(
	pubKeyHash []byte,
	amount int,
) (map[string][]int, int) {
	if bc == nil {
		return nil, 0
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return bc.FindSpendableOutputs(
		pubKeyHash,
		amount,
	)
}

// FindUTXOsSnapshot returns detached mature outputs from one coherent
// canonical UTXO view. Legacy adapters must use this boundary instead of
// owning or exposing another mutable UTXO map.
func (bc *Blockchain) FindUTXOsSnapshot(
	pubKeyHash []byte,
) []TransactionOutput {
	result := []TransactionOutput{}
	if bc == nil || len(pubKeyHash) == 0 {
		return result
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	if bc.UTXO == nil {
		bc.UpdateUTXOSet()
	}

	for txID, outputs := range bc.UTXO {
		rawTxID, err := hex.DecodeString(txID)
		if err != nil {
			continue
		}

		for outIndex, output := range outputs {
			if output.Amount <= 0 {
				continue
			}

			if !bc.isCoinbaseOutputMature(rawTxID, outIndex) {
				continue
			}

			if !output.IsLockedWithKey(pubKeyHash) {
				continue
			}

			result = append(
				result,
				cloneTransactionOutputV1(output),
			)
		}
	}

	return result
}

// ValidateBlockchainSynchronized protects complete live-chain validation.
func (bc *Blockchain) ValidateBlockchainSynchronized() error {
	if bc == nil {
		return ErrNilBlockchain
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return bc.ValidateBlockchain()
}

// BindCanonicalActivationPolicySynchronizedV1 serializes mutation of the
// canonical activation-policy binding.
func (bc *Blockchain) BindCanonicalActivationPolicySynchronizedV1() error {
	policy, err := qconsensus.LoadCanonicalActivationPolicyV1()
	if err != nil {
		return err
	}
	return bc.BindActivationPolicySynchronizedV1(policy)
}

// BindActivationPolicySynchronizedV1 retains the existing one-time binder
// while serializing the publication of a verified policy.
func (bc *Blockchain) BindActivationPolicySynchronizedV1(policy qconsensus.ActivationPolicyV1) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return bc.BindActivationPolicyV1(policy)
}

// ValidatePeerBlockSynchronized protects state-dependent strict peer-block
// validation.
func (bc *Blockchain) ValidatePeerBlockSynchronized(
	block *Block,
) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return bc.ValidatePeerBlockStrictV2(block)
}

// ValidatePeerTransactionSynchronized protects strict peer transaction
// validation and legacy lazy UTXO state initialization.
func (bc *Blockchain) ValidatePeerTransactionSynchronized(
	tx *Transaction,
) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return bc.ValidatePeerTransactionStrictV1(tx)
}

// ValidateReplacementChainSynchronized validates a replacement candidate
// against one coherent local Blockchain state.
func (bc *Blockchain) ValidateReplacementChainSynchronized(
	blocks []*Block,
) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return bc.ValidateReplacementChainStrictV1(blocks)
}

// UpdateUTXOSetSynchronized is the external synchronization boundary for
// explicit UTXO rebuild requests.
func (bc *Blockchain) UpdateUTXOSetSynchronized() {
	if bc == nil {
		return
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	bc.UpdateUTXOSet()
}

// IsValidChainSynchronized protects validation that traverses live Blocks.
func (bc *Blockchain) IsValidChainSynchronized() bool {
	if bc == nil {
		return false
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return bc.IsValidChain()
}

// AddBlockFromPeerSynchronizedV2 synchronizes peer-facing strict validation.
// The Blockchain-owned commit path acquires the canonical writer lock before
// mutation and rechecks its commit conditions.
func (bc *Blockchain) AddBlockFromPeerSynchronizedV2(
	block *Block,
) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	if err := bc.ValidatePeerBlockSynchronized(block); err != nil {
		logPeerBlockReject(
			"BLOCK_PEER_STRICT_VALIDATION_FAILED",
			block,
			err,
		)
		return err
	}

	return bc.addBlockFromPeerAfterStrictV2(block)
}

// AddTransactionFromPeerSynchronizedV1 protects peer validation. AddTransaction
// owns the final canonical writer boundary before mempool insertion.
func (bc *Blockchain) AddTransactionFromPeerSynchronizedV1(
	tx *Transaction,
) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	if err := bc.ValidatePeerTransactionSynchronized(tx); err != nil {
		logMempoolReject(
			"PEER_TX_STRICT_VALIDATION_FAILED",
			tx,
			err,
		)
		return err
	}

	return bc.AddTransaction(tx)
}

// ReplaceChainFromPeerSynchronizedV1 protects peer-facing replacement
// validation. The authoritative replacement path owns the writer boundary.
func (bc *Blockchain) ReplaceChainFromPeerSynchronizedV1(
	blocks []*Block,
) error {
	if bc == nil {
		return ErrNilBlockchain
	}

	if err := bc.ValidateReplacementChainSynchronized(blocks); err != nil {
		return err
	}

	return bc.replaceChainAfterStrictV1(blocks)
}

// BlocksSnapshot returns a detached copy of the canonical chain.
// The returned blocks can be serialized without exposing live Blockchain state.
func (bc *Blockchain) BlocksSnapshot() []*Block {
	if bc == nil {
		return nil
	}

	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()

	return cloneBlocksV1(bc.Blocks)
}

// FindSpendableOutputsAtHeightSnapshotV1 protects denomination-aware
// transaction construction from concurrent canonical state mutation.
func (bc *Blockchain) FindSpendableOutputsAtHeightSnapshotV1(
	pubKeyHash []byte,
	amount int,
	spendHeight int,
) (map[string][]int, int, error) {
	if bc == nil {
		return nil, 0, ErrChainNotInitialized
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return bc.FindSpendableOutputsAtHeightV1(
		pubKeyHash,
		amount,
		spendHeight,
	)
}

// ActivationPolicySnapshotV1 returns an independent value for trusted runtime
// reload/peer validation. No serialized input can supply its private proof.
func (bc *Blockchain) ActivationPolicySnapshotV1() qconsensus.ActivationPolicyV1 {
	if bc == nil {
		return qconsensus.ActivationPolicyV1{}
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	return bc.activationPolicy.policy
}
