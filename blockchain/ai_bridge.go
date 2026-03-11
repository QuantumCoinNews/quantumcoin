// blockchain/ai_bridge.go
package blockchain

import (
	"encoding/hex"

	"quantumcoin/ai"
)

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
