package blockchain

import "fmt"

// PEER_TRANSACTION_VALIDATION_V1
//
// This file hardens transactions received from peers.
// It does not mint, transfer, change rewards, or change consensus policy by itself.
// It only rejects unsafe peer-supplied transactions before they enter the chain mempool.

// AddTransactionFromPeerStrictV1 validates a peer transaction with explicit
// peer-facing checks and then delegates to the canonical AddTransaction path.
func (bc *Blockchain) AddTransactionFromPeerStrictV1(tx *Transaction) error {
	if err := bc.ValidatePeerTransactionStrictV1(tx); err != nil {
		logMempoolReject("PEER_TX_STRICT_VALIDATION_FAILED", tx, err)
		return err
	}

	return bc.AddTransaction(tx)
}

// ValidatePeerTransactionStrictV1 makes the P2P transaction acceptance rule explicit.
// The canonical AddTransaction path still performs the final validation again.
func (bc *Blockchain) ValidatePeerTransactionStrictV1(tx *Transaction) error {
	if bc == nil {
		return fmt.Errorf("peer tx strict validation: nil blockchain")
	}
	if tx == nil {
		return ErrNilTransaction
	}

	if err := ValidateTransactionBasic(tx); err != nil {
		return fmt.Errorf("peer tx basic validation failed: %w", err)
	}

	if tx.IsCoinbase() {
		return fmt.Errorf("peer tx coinbase is not allowed in mempool")
	}

	if len(tx.ID) == 0 {
		return fmt.Errorf("peer tx missing id")
	}
	if len(tx.Inputs) == 0 {
		return fmt.Errorf("peer tx has no inputs")
	}
	if len(tx.Outputs) == 0 {
		return fmt.Errorf("peer tx has no outputs")
	}

	if err := peerTxVerifyNoPanicV1(tx); err != nil {
		return err
	}

	if bc.UTXO == nil {
		bc.UpdateUTXOSet()
	}

	if err := bc.validateTransactionSpend(tx); err != nil {
		return fmt.Errorf("peer tx spend validation failed: %w", err)
	}

	return nil
}

// peerTxVerifyNoPanicV1 converts malformed peer transaction panics into normal
// reject errors. Peer input is untrusted and must never be able to crash the node.
func peerTxVerifyNoPanicV1(tx *Transaction) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("peer tx invalid signature: %v", r)
		}
	}()

	if !tx.Verify() {
		return fmt.Errorf("peer tx invalid signature")
	}

	return nil
}
