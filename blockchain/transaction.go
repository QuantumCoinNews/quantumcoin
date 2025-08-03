package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"quantumcoin/wallet"
)

// TransactionInput: UTXO modeli için giriş
type TransactionInput struct {
	TxID      []byte
	OutIndex  int
	Signature []byte
	PubKey    []byte
}

// TransactionOutput: UTXO modeli için çıkış
type TransactionOutput struct {
	Amount     int
	PubKeyHash []byte
}

// Transaction: Ana işlem tipi
type Transaction struct {
	ID        []byte
	Inputs    []TransactionInput
	Outputs   []TransactionOutput
	Timestamp time.Time // ⏰ İşlem zamanı
	Sender    string    // 👤 Gönderen cüzdan adresi
	Amount    float64   // 💸 Toplam gönderim miktarı (kullanım kolaylığı için)
}

// NewTransaction: Yeni transfer işlemi oluşturur
func NewTransaction(from string, to string, amount int, bc *Blockchain) (*Transaction, error) {
	w := wallet.LoadWallet(from)
	pubKeyHash := wallet.GetPubKeyHash(w.PublicKey)

	utxos, acc := bc.FindSpendableOutputs(pubKeyHash, amount)
	if acc < amount {
		return nil, fmt.Errorf("yetersiz bakiye")
	}

	var inputs []TransactionInput
	for txid, outs := range utxos {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			return nil, err
		}
		for _, outIdx := range outs {
			input := TransactionInput{
				TxID:      txID,
				OutIndex:  outIdx,
				Signature: nil,
				PubKey:    w.PublicKey,
			}
			inputs = append(inputs, input)
		}
	}

	var outputs []TransactionOutput
	// Alıcıya gönderim
	outputs = append(outputs, TransactionOutput{
		Amount:     amount,
		PubKeyHash: wallet.Base58DecodeAddress(to),
	})

	// Para üstü, varsa
	if acc > amount {
		outputs = append(outputs, TransactionOutput{
			Amount:     acc - amount,
			PubKeyHash: pubKeyHash,
		})
	}

	tx := &Transaction{
		ID:        nil,
		Inputs:    inputs,
		Outputs:   outputs,
		Timestamp: time.Now(),      // 🟢 ŞİMDİ ZAMANI
		Sender:    from,            // 🟢 Gönderen cüzdan
		Amount:    float64(amount), // 🟢 İşlem tutarı
	}
	tx.ID = tx.Hash()
	return tx, nil
}

// Hash: İşlemi hash'ler (SHA-256)
func (tx *Transaction) Hash() []byte {
	var hash [32]byte
	txCopy := *tx
	txCopy.ID = []byte{}
	data := txCopy.Serialize()
	hash = sha256.Sum256(data)
	return hash[:]
}

// Serialize: İşlemi []byte'a çevirir (gob)
func (tx *Transaction) Serialize() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(tx)
	if err != nil {
		log.Panicf("İşlem serileştirme hatası: %v", err)
	}
	return buff.Bytes()
}

// Verify: İşlemi doğrular (geliştirilebilir)
func (tx *Transaction) Verify() bool {
	return tx != nil && len(tx.Outputs) > 0
}

// IsCoinbase: Blok ödülü işlemi mi?
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 0
}

// IsLockedWithKey: Çıkış belirli adrese mi kilitli?
func (out *TransactionOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

// UsesKey: Input belirli bir pubkey ile harcanabilir mi?
func (in *TransactionInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.HashPubKey(in.PubKey)
	return bytes.Equal(lockingHash, pubKeyHash)
}
