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
	for _, block := range chain.Blocks {
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
			// Tüm çıkışları geçici ekle
			for _, out := range tx.Outputs {
				set.UTXOs[txID] = append(set.UTXOs[txID], out)
			}
			// Eğer coinbase değilse girişlere göre harcananları sil
			if len(tx.Inputs) > 0 {
				for _, in := range tx.Inputs {
					inTxID := hex.EncodeToString(in.TxID)
					index := in.OutIndex
					outs := set.UTXOs[inTxID]
					if index < len(outs) {
						set.UTXOs[inTxID] = append(outs[:index], outs[index+1:]...)
					}
					if len(set.UTXOs[inTxID]) == 0 {
						delete(set.UTXOs, inTxID)
					}
				}
			}
		}
	}
}

// FindSpendableOutputs: Belirli bir miktarı sağlayacak UTXO'ları bulur
func (set *UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	accumulated := 0
	spendable := make(map[string][]int)
	for txID, outs := range set.UTXOs {
		for idx, out := range outs {
			if out.IsLockedWithKey(pubKeyHash) {
				accumulated += out.Amount
				spendable[txID] = append(spendable[txID], idx)
				if accumulated >= amount {
					return accumulated, spendable
				}
			}
		}
	}
	return accumulated, spendable
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
