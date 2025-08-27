package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"time"

	"quantumcoin/wallet"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// --------- TX Yapıları ---------

type TransactionInput struct {
	TxID      []byte // Harcanan çıktının TxID'si
	OutIndex  int    // Hangi output
	Signature []byte // encodeSig(r,s)
	PubKey    []byte // Uncompressed (0x04||X||Y)
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
	// PubKey boşsa true (eski davranış) — asıl doğrulama Verify()'da
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

func pad32(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// encode/decode ECDSA imzası (r,s)
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

// Uncompressed (0x04||X||Y) -> *ecdsa.PublicKey (secp256k1)
func pubKeyToECDSA(pub []byte) *ecdsa.PublicKey {
	if len(pub) != 65 || pub[0] != 0x04 {
		return nil
	}
	curve := secp256k1.S256()
	x := new(big.Int).SetBytes(pub[1:33])
	y := new(big.Int).SetBytes(pub[33:65])
	if !curve.IsOnCurve(x, y) {
		return nil
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
}

// İmza mesajı (deterministik): TrimmedCopy.Serialize + input.TxID + input.OutIndex
func signMessageBytes(tx *Transaction, inputIdx int) []byte {
	txCopy := tx.TrimmedCopy()
	var buf bytes.Buffer
	buf.Write(txCopy.Serialize())

	in := tx.Inputs[inputIdx]
	buf.Write(in.TxID)

	var outIdx [4]byte
	binary.BigEndian.PutUint32(outIdx[:], uint32(in.OutIndex))
	buf.Write(outIdx[:])

	sum := sha256.Sum256(buf.Bytes())
	return sum[:]
}

// --------- İşlem oluşturma / imzalama / doğrulama ---------

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

// Tüm input'ları imzala (secp256k1)
func (tx *Transaction) Sign(priv *ecdsa.PrivateKey) error {
	if tx == nil {
		return fmt.Errorf("nil tx")
	}
	if tx.IsCoinbase() {
		return nil
	}

	fromPKH := wallet.Base58DecodeAddress(tx.Sender)
	if len(fromPKH) == 0 {
		return fmt.Errorf("invalid sender address")
	}

	// Uncompressed pubkey (65B)
	pub := append([]byte{0x04}, pad32(priv.PublicKey.X.Bytes())...)
	pub = append(pub, pad32(priv.PublicKey.Y.Bytes())...)

	for i := range tx.Inputs {
		msg := signMessageBytes(tx, i)

		r, s, err := ecdsa.Sign(rand.Reader, priv, msg)
		if err != nil {
			return fmt.Errorf("ecdsa sign failed: %w", err)
		}
		tx.Inputs[i].Signature = encodeSig(r, s)
		tx.Inputs[i].PubKey = pub
	}
	return nil
}

// İmzaları doğrula (coinbase hariç)
func (tx *Transaction) Verify() bool {
	if tx == nil {
		return false
	}
	if tx.IsCoinbase() {
		return len(tx.Outputs) > 0
	}
	if len(tx.Inputs) == 0 || len(tx.Outputs) == 0 {
		return false
	}

	fromPKH := wallet.Base58DecodeAddress(tx.Sender)
	if len(fromPKH) == 0 {
		return false
	}

	for i := range tx.Inputs {
		in := tx.Inputs[i]
		if len(in.Signature) == 0 || len(in.PubKey) != 65 || in.PubKey[0] != 0x04 {
			return false
		}
		if !in.UsesKey(fromPKH) {
			return false
		}
		pub := pubKeyToECDSA(in.PubKey)
		if pub == nil {
			return false
		}
		r, s := decodeSig(in.Signature)
		if r == nil || s == nil {
			return false
		}
		msg := signMessageBytes(tx, i)
		if !ecdsa.Verify(pub, msg, r, s) {
			return false
		}
	}
	return true
}

// ---- Web cüzdan için: her input’un imzalanacak mesajının HEX’i ----
func SigningHashes(tx *Transaction) []string {
	out := make([]string, 0, len(tx.Inputs))
	for i := range tx.Inputs {
		h := signMessageBytes(tx, i)
		out = append(out, hex.EncodeToString(h))
	}
	return out
}
