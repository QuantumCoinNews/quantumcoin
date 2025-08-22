package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"time"

	"quantumcoin/wallet"
)

// --------- TX Yapıları ---------

type TransactionInput struct {
	TxID      []byte // Harcanan çıktının TxID'si
	OutIndex  int    // Hangi output
	Signature []byte // r||s (isteğe bağlı)
	PubKey    []byte // Uncompressed (0x04||X||Y) (isteğe bağlı)
}

type TransactionOutput struct {
	Amount     int
	PubKeyHash []byte // Hash160(pubkey)
}

type Transaction struct {
	ID        []byte
	Inputs    []TransactionInput
	Outputs   []TransactionOutput
	Timestamp time.Time
	Sender    string  // kolaylık alanı (adres string)
	Amount    float64 // kolaylık alanı
}

// --------- Yardımcılar ---------

func (out *TransactionOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

func (in *TransactionInput) UsesKey(pubKeyHash []byte) bool {
	// Basit profil: PubKey yoksa true kabul et (doğrulama zorunlu değil)
	if len(in.PubKey) == 0 {
		return true
	}
	lockingHash := wallet.HashPubKey(in.PubKey)
	return bytes.Equal(lockingHash, pubKeyHash)
}

func (tx *Transaction) IsCoinbase() bool { return len(tx.Inputs) == 0 }

func (tx *Transaction) Serialize() []byte {
	var buff bytes.Buffer
	if err := gob.NewEncoder(&buff).Encode(tx); err != nil {
		log.Panicf("tx serialize error: %v", err)
	}
	return buff.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var h [32]byte
	copyTx := *tx
	copyTx.ID = nil
	h = sha256.Sum256(copyTx.Serialize())
	return h[:]
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TransactionInput
	for _, in := range tx.Inputs {
		inputs = append(inputs, TransactionInput{
			TxID:      append([]byte(nil), in.TxID...),
			OutIndex:  in.OutIndex,
			Signature: nil,
			PubKey:    nil,
		})
	}
	outs := make([]TransactionOutput, len(tx.Outputs))
	copy(outs, tx.Outputs)
	return Transaction{
		ID:        append([]byte(nil), tx.ID...),
		Inputs:    inputs,
		Outputs:   outs,
		Timestamp: tx.Timestamp,
		Sender:    tx.Sender,
		Amount:    tx.Amount,
	}
}

// encode/decode ECDSA imzası (opsiyonel kullanılır)
func encodeSig(r, s *big.Int) []byte {
	rb := r.Bytes()
	sb := s.Bytes()
	out := make([]byte, 0, len(rb)+len(sb)+2)
	out = append(out, byte(len(rb)))
	out = append(out, rb...)
	out = append(out, byte(len(sb)))
	out = append(out, sb...)
	return out
}
func decodeSig(sig []byte) (r, s *big.Int) {
	if len(sig) < 2 {
		return new(big.Int), new(big.Int)
	}
	rl := int(sig[0])
	if 1+rl >= len(sig) {
		return new(big.Int), new(big.Int)
	}
	r = new(big.Int).SetBytes(sig[1 : 1+rl])
	sb := sig[1+rl:]
	if len(sb) < 1 {
		return r, new(big.Int)
	}
	sl := int(sb[0])
	if 1+sl > len(sb) {
		return r, new(big.Int)
	}
	s = new(big.Int).SetBytes(sb[1 : 1+sl])
	return r, s
}
func pubKeyToECDSA(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 || pub[0] != 0x04 {
		return nil
	}
	curve := elliptic.P256()
	byteLen := (curve.Params().BitSize + 7) / 8
	if len(pub) != 1+2*byteLen {
		return nil
	}
	x := new(big.Int).SetBytes(pub[1 : 1+byteLen])
	y := new(big.Int).SetBytes(pub[1+byteLen:])
	if !curve.IsOnCurve(x, y) {
		return nil
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
}

func NewTransaction(from string, to string, amount int, bc *Blockchain) (*Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	pubFrom := wallet.Base58DecodeAddress(from)

	utxos, acc := bc.FindSpendableOutputs(pubFrom, amount)
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
			inputs = append(inputs, TransactionInput{
				TxID:     txID,
				OutIndex: outIdx,
			})
		}
	}

	var outputs []TransactionOutput
	// Alıcı
	outputs = append(outputs, TransactionOutput{
		Amount:     amount,
		PubKeyHash: wallet.Base58DecodeAddress(to),
	})
	// Para üstü
	if acc > amount {
		outputs = append(outputs, TransactionOutput{
			Amount:     acc - amount,
			PubKeyHash: pubFrom,
		})
	}

	tx := &Transaction{
		ID:        nil,
		Inputs:    inputs,
		Outputs:   outputs,
		Timestamp: time.Now(),
		Sender:    from,
		Amount:    float64(amount),
	}
	tx.ID = tx.Hash()
	return tx, nil
}

func (tx *Transaction) Sign(_ *ecdsa.PrivateKey) error { return nil }

func (tx *Transaction) Verify() bool {
	// Basit profil: geçerli format kontrolü
	if tx == nil {
		return false
	}
	if tx.IsCoinbase() {
		return len(tx.Outputs) > 0
	}
	return len(tx.Inputs) > 0 && len(tx.Outputs) > 0
}
