// blockchain/ai_bridge.go

// Package blockchain contains QuantumCoin project code for this module.
package blockchain

import (
	"encoding/hex"

	"quantumcoin/ai"
	qconsensus "quantumcoin/consensus"
)

// ToAITxLite documents the QuantumCoin component used by this module.
// blockchain.Transaction -> ai.TxLite
func ToAITxLite(txs []*Transaction) []ai.TxLite {
	out := make([]ai.TxLite, 0, len(txs))
	for _, tx := range txs {
		if tx == nil {
			continue
		}
		addr := tx.Sender
		out = append(out, ai.TxLite{
			TxID:          hex.EncodeToString(tx.ID),
			WalletAddress: addr,
			Sender:        tx.Sender,
			Amount:        tx.Amount,
			Timestamp:     tx.Timestamp,
		})
	}
	return out
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

// ToAITxLiteAtHeightV1 converts confirmed transaction amounts into QC for
// advisory AI analysis. The block's origin height determines the wire unit;
// it must not be inferred from the current chain tip.
func (bc *Blockchain) ToAITxLiteAtHeightV1(
	txs []*Transaction,
	height int,
) []ai.TxLite {
	if bc == nil {
		return nil
	}

	out := ToAITxLite(txs)
	for i := range out {
		out[i].Amount = bc.TransactionAmountQCAtHeightV1(out[i].Amount, height)
	}
	return out
}
