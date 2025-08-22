// blockchain/tx.go
package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex" // txid'leri gerçek bayta çevirmek için
	"errors"
	"time"

	"quantumcoin/wallet"
)

// TransactionInput, bir işlemin harcanan çıktısını (UTXO) referanslar.
type TransactionInput struct {
	// TxID, harcanan çıktının ait olduğu işlem ID'sinin ham baytlarıdır.
	TxID     []byte
	OutIndex int // o işlemin kaçıncı çıktısı
	// İmza alanları şimdilik boş; demo için doğrulama gevşek tutuldu
	Signature []byte
	PubKey    []byte
}

// TransactionOutput, bir alıcıya tahsis edilen miktarı ve kilit koşulunu taşır.
type TransactionOutput struct {
	Amount     int
	PubKeyHash []byte // QC adresindeki HASH160 (versiyonsuz)
}

// IsLockedWithKey, output'un verilen pubKeyHash ile kilitli olup olmadığını döndürür.
func (out *TransactionOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

// Transaction, bir QC transferini temsil eder.
type Transaction struct {
	ID        []byte
	Inputs    []TransactionInput
	Outputs   []TransactionOutput
	Timestamp time.Time
	Sender    string  // gösterim/diagnostic
	Amount    float64 // gösterim/diagnostic
}

// Serialize, işlemi gob ile bayt dizisine çevirir.
func (tx *Transaction) Serialize() []byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(tx); err != nil {
		// İstersen burada log/panic tercih edebilirsin.
		return nil
	}
	return buf.Bytes()
}

// DeserializeTransaction, gob ile serileştirilmiş baytları Transaction'a çevirir.
func DeserializeTransaction(b []byte) *Transaction {
	var tx Transaction
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&tx); err != nil {
		return nil
	}
	return &tx
}

// Hash, işlem ID'sini (txid) hesaplar. ID alanını hariç tutar.
func (tx *Transaction) Hash() []byte {
	cp := *tx
	cp.ID = nil
	sum := sha256.Sum256(cp.Serialize())
	return sum[:]
}

// IsCoinbase, işlemin coinbase (input'u olmayan) olup olmadığını döndürür.
func (tx *Transaction) IsCoinbase() bool { return len(tx.Inputs) == 0 }

// Verify, demo amaçlı gevşek bir doğrulama yapar (imza zorunlu değil).
func (tx *Transaction) Verify() bool {
	// İlerde: her input için ECDSA imza kontrolü
	return true
}

// spendableFinder, Blockchain bağımlılığını gevşetmek için gereken minimum arayüzdür.
// *Blockchain bu metodu zaten sağladığı için çağıran tarafta değişiklik gerekmez.
type spendableFinder interface {
	FindSpendableOutputs(pubKeyHash []byte, amount int) (map[string][]int, int)
}

// NewTransaction, 'from' adresinden 'to' adresine 'amount' QC gönderir.
// Not: İmza zorunlu tutulmuyor; UTXO seçimleriyle harcama yapılır.
func NewTransaction(from, to string, amount int, bc spendableFinder) (*Transaction, error) {
	if bc == nil {
		return nil, errors.New("blockchain is nil")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	fromPKH := wallet.Base58DecodeAddress(from)
	toPKH := wallet.Base58DecodeAddress(to)

	accum, spendable := 0, map[string][]int{}
	// bc.FindSpendableOutputs(pubKeyHash, amount) imzası (map, int)
	spendable, accum = bc.FindSpendableOutputs(fromPKH, amount)
	if accum < amount {
		return nil, errors.New("insufficient funds")
	}

	var inputs []TransactionInput
	var outputs []TransactionOutput

	// Seçilen UTXO'ları input yap
	for txIDHex, idxs := range spendable {
		// hex string -> gerçek txid baytları
		txid, err := hex.DecodeString(txIDHex)
		if err != nil {
			return nil, errors.New("invalid txid hex in spendable set")
		}
		for _, outIdx := range idxs {
			inputs = append(inputs, TransactionInput{
				TxID:      txid,
				OutIndex:  outIdx,
				Signature: nil,
				PubKey:    nil,
			})
		}
	}

	// Alıcıya output
	outputs = append(outputs, TransactionOutput{
		Amount:     amount,
		PubKeyHash: toPKH,
	})

	// Change (para üstü) varsa geri gönder
	if change := accum - amount; change > 0 {
		outputs = append(outputs, TransactionOutput{
			Amount:     change,
			PubKeyHash: fromPKH,
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
