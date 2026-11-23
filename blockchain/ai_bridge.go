// blockchain/ai_bridge.go

// Package blockchain contains QuantumCoin project code for this module.
package blockchain

import (
	"bytes"
	"encoding/hex"
	"time"

	"quantumcoin/ai"
	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

// verifiedAIProjectionV1 is used only for locked canonical snapshots. Signed
// wire convenience fields are not measurements of value or observation time.
func (bc *Blockchain) verifiedAIProjectionV1(tx *Transaction, height int, observedAt time.Time) (ai.TxLite, bool) {
	if observedAt.IsZero() || tx.IsCoinbase() || !wallet.ValidateAddress(tx.Sender) ||
		ValidateTransactionBasic(tx) != nil || !bc.verifyTransactionLockedV2(tx) {
		return ai.TxLite{}, false
	}
	address := wallet.GetAddressFromPub(tx.Inputs[0].PubKey)
	if address != tx.Sender {
		return ai.TxLite{}, false
	}
	change := wallet.HashPubKey(tx.Inputs[0].PubKey)
	var amount int64
	for _, out := range tx.Outputs {
		if !bytes.Equal(out.PubKeyHash, change) {
			amount += int64(out.Amount)
		}
	}
	return ai.TxLite{
		TxID: hex.EncodeToString(tx.ID), Sender: address, WalletAddress: address,
		Amount: bc.TransactionAmountQCAtHeightV1(float64(amount), height), Timestamp: observedAt,
	}, true
}

// TransactionAmountQCAtHeightV1 is the shared advisory QC presentation
// boundary for transaction Amount, which carries protocol units on the wire.
func (bc *Blockchain) TransactionAmountQCAtHeightV1(amount float64, height int) float64 {
	if bc == nil {
		return 0
	}
	if qconsensus.ProtocolUsesAtomDenominationV1(bc.ProtocolAtHeightV1(height)) {
		return amount / float64(qconsensus.CanonicalAtomsPerQCV1)
	}
	return amount
}
