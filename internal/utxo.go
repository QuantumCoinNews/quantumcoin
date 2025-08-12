package internal

import (
	"encoding/hex"

	"quantumcoin/blockchain"
)

// UTXOSet: Zincirdeki harcanmamış tüm çıktıları tutar
type UTXOSet struct {
	UTXOs map[string][]blockchain.TransactionOutput
}

// NewUTXOSet: Yeni boş UTXO seti oluşturur
func NewUTXOSet() *UTXOSet {
	return &UTXOSet{UTXOs: make(map[string][]blockchain.TransactionOutput)}
}

// Reindex: Zinciri baştan tarar ve UTXO'ları günceller
func (set *UTXOSet) Reindex(chain *blockchain.Blockchain) {
	set.UTXOs = make(map[string][]blockchain.TransactionOutput)

	// 1) Tüm çıktıları ekle
	for _, b := range chain.Blocks {
		for _, tx := range b.Transactions {
			txID := hex.EncodeToString(tx.ID)
			if _, ok := set.UTXOs[txID]; !ok {
				set.UTXOs[txID] = make([]blockchain.TransactionOutput, 0, len(tx.Outputs))
			}
			set.UTXOs[txID] = append(set.UTXOs[txID], tx.Outputs...)
		}
	}

	// 2) Non-coinbase işlemlerin harcadığı çıktıları düş
	for _, b := range chain.Blocks {
		for _, tx := range b.Transactions {
			if tx.IsCoinbase() {
				continue
			}
			for _, in := range tx.Inputs {
				inTxID := hex.EncodeToString(in.TxID)
				outs := set.UTXOs[inTxID]
				idx := in.OutIndex
				if idx >= 0 && idx < len(outs) {
					outs = append(outs[:idx], outs[idx+1:]...)
					if len(outs) == 0 {
						delete(set.UTXOs, inTxID)
					} else {
						set.UTXOs[inTxID] = outs
					}
				}
			}
		}
	}
}

// FindSpendableOutputs: Belirli bir miktarı sağlayacak UTXO'ları bulur
// İMZA UYUMU: blockchain.FindSpendableOutputs ile aynı sırada döndürür (map, int)
func (set *UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (map[string][]int, int) {
	accumulated := 0
	spendable := make(map[string][]int)

	for txID, outs := range set.UTXOs {
		for idx, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				accumulated += out.Amount
				spendable[txID] = append(spendable[txID], idx)
				if accumulated >= amount {
					return spendable, accumulated
				}
			}
		}
	}
	return spendable, accumulated
}

// FindUTXOs: Adrese ait tüm harcanmamış çıktıları döner
func (set *UTXOSet) FindUTXOs(pubKeyHash []byte) []blockchain.TransactionOutput {
	var utxos []blockchain.TransactionOutput
	for _, outs := range set.UTXOs {
		for _, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				utxos = append(utxos, out)
			}
		}
	}
	return utxos
}
