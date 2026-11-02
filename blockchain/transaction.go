package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
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
	if out == nil || IsCanonicalBurnOutputV1(*out) {
		return false
	}
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

func (in *TransactionInput) UsesKey(pubKeyHash []byte) bool {
	// PubKey boşsa true (eski davranış) — asıl doğrulama Verify()'da
	if len(in.PubKey) == 0 {
		return true
	}

	pub := in.PubKey
	// Wallet adresleri 64 byte X||Y public key üzerinden üretiliyor.
	// TX imza alanında ise standart uncompressed format 0x04||X||Y saklanıyor.
	// Adres hash kontrolünde 0x04 prefix'i atılmazsa doğru private key bile
	// "verify failed" verir.
	if len(pub) == 65 && pub[0] == 0x04 {
		pub = pub[1:]
	}

	lockingHash := wallet.HashPubKey(pub)
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
	id, err := CanonicalTransactionIDV1(tx)
	if err != nil {
		log.Panicf("tx hash error: %v", err)
	}
	return id
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

// İmza mesajı (deterministik):
// TrimmedCopy.Serialize + input.TxID + input.OutIndex.
//
// Tarihsel gob serialization blockchain paketinde korunur.
// Digest hesabı bağımsız consensus katmanına yönlendirilir.
func signMessageBytes(
	tx *Transaction,
	inputIdx int,
) []byte {
	txCopy := tx.TrimmedCopy()
	txCopy.ID = nil

	trimmedPayload := txCopy.Serialize()
	input := tx.Inputs[inputIdx]

	digest, err := qconsensus.SigningDigestV1(
		trimmedPayload,
		input.TxID,
		input.OutIndex,
	)
	if err != nil {
		log.Panicf(
			"transaction signing digest error: %v",
			err,
		)
	}

	return append(
		[]byte(nil),
		digest[:]...,
	)
}

// --------- İşlem oluşturma / imzalama / doğrulama ---------

func NewTransaction(
	from string,
	to string,
	amount int,
	bc *Blockchain,
) (*Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	if bc == nil {
		return nil, fmt.Errorf("nil blockchain")
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	spendHeight := bc.nextSpendHeightV1()

	amountUnits, err :=
		bc.wholeQCAmountToTransactionUnitsAtHeightV1(
			amount,
			spendHeight,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"convert whole-QC transaction amount: %w",
			err,
		)
	}

	return newTransactionAtHeightLockedV1(
		from,
		to,
		amountUnits,
		spendHeight,
		bc,
	)
}

// NewTransactionInProtocolUnitsV1 creates a transaction whose amount is
// already expressed in the integer monetary unit required by the candidate
// protocol height.
//
// LegacyV1 callers pass whole-QC integers.
// AtomV3 callers pass atom integers.
//
// User-facing decimal QC parsing must happen before this boundary.
// NewTransactionFromQCAmountV1 creates a transaction from an exact
// user-facing QC decimal string. No floating-point conversion is used.
func NewTransactionFromQCAmountV1(
	from string,
	to string,
	rawQC string,
	bc *Blockchain,
) (*Transaction, error) {
	if bc == nil {
		return nil, fmt.Errorf("nil blockchain")
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	spendHeight := bc.nextSpendHeightV1()

	amountUnits, err :=
		bc.qcAmountStringToTransactionUnitsAtHeightV1(
			rawQC,
			spendHeight,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"convert exact QC transaction amount: %w",
			err,
		)
	}

	return newTransactionAtHeightLockedV1(
		from,
		to,
		amountUnits,
		spendHeight,
		bc,
	)
}

func NewTransactionInProtocolUnitsV1(
	from string,
	to string,
	amountUnits int,
	bc *Blockchain,
) (*Transaction, error) {
	if amountUnits <= 0 {
		return nil, fmt.Errorf("invalid protocol-unit amount")
	}
	if bc == nil {
		return nil, fmt.Errorf("nil blockchain")
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	spendHeight := bc.nextSpendHeightV1()

	return newTransactionAtHeightLockedV1(
		from,
		to,
		amountUnits,
		spendHeight,
		bc,
	)
}

func newTransactionAtHeightV1(
	from string,
	to string,
	amountUnits int,
	spendHeight int,
	bc *Blockchain,
) (*Transaction, error) {
	if bc == nil {
		return nil, fmt.Errorf("nil blockchain")
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	return newTransactionAtHeightLockedV1(
		from,
		to,
		amountUnits,
		spendHeight,
		bc,
	)
}

// newTransactionAtHeightLockedV1 requires bc.stateMu to be held by the caller.
// Candidate height, denomination conversion and UTXO selection therefore use
// one coherent canonical-state snapshot.
func newTransactionAtHeightLockedV1(
	from string,
	to string,
	amountUnits int,
	spendHeight int,
	bc *Blockchain,
) (*Transaction, error) {
	if bc == nil {
		return nil, fmt.Errorf("nil blockchain")
	}
	if amountUnits <= 0 {
		return nil, fmt.Errorf("invalid protocol-unit amount")
	}
	if spendHeight < 0 {
		return nil, fmt.Errorf(
			"invalid spend height %d",
			spendHeight,
		)
	}

	pubFrom := wallet.Base58DecodeAddress(from)

	utxos, acc, err :=
		bc.FindSpendableOutputsAtHeightV1(
			pubFrom,
			amountUnits,
			spendHeight,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"find spendable outputs: %w",
			err,
		)
	}

	if acc < amountUnits {
		return nil, fmt.Errorf("yetersiz bakiye")
	}

	var inputs []TransactionInput

	for txid, outs := range utxos {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			return nil, err
		}

		for _, outIdx := range outs {
			inputs = append(
				inputs,
				TransactionInput{
					TxID:     txID,
					OutIndex: outIdx,
				},
			)
		}
	}

	var outputs []TransactionOutput

	outputs = append(
		outputs,
		TransactionOutput{
			Amount:     amountUnits,
			PubKeyHash: wallet.Base58DecodeAddress(to),
		},
	)

	if acc > amountUnits {
		outputs = append(
			outputs,
			TransactionOutput{
				Amount:     acc - amountUnits,
				PubKeyHash: pubFrom,
			},
		)
	}

	tx := &Transaction{
		ID:        nil,
		Inputs:    inputs,
		Outputs:   outputs,
		Timestamp: time.Now(),
		Sender:    from,
		Amount:    float64(amountUnits),
	}

	tx.ID = tx.Hash()

	return tx, nil
}

// Sign documents the QuantumCoin component used by this module.
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

	// Legacy uncompressed public key biçimi korunuyor:
	// 0x04 || 32-byte X || 32-byte Y.
	//
	// Byte üretimi bağımsız consensus katmanından yapılır.
	pub, err := qconsensus.UncompressedPublicKeyV1(priv)
	if err != nil {
		return fmt.Errorf(
			"serialize transaction public key: %w",
			err,
		)
	}
	for i := range tx.Inputs {
		message := signMessageBytes(tx, i)

		var digest [32]byte
		if len(message) != len(digest) {
			return fmt.Errorf(
				"transaction signing digest length is %d, want %d",
				len(message),
				len(digest),
			)
		}
		copy(digest[:], message)

		signature, err :=
			qconsensus.SignTransactionDigestV1(
				priv,
				digest,
			)
		if err != nil {
			return fmt.Errorf(
				"ecdsa sign failed: %w",
				err,
			)
		}

		tx.Inputs[i].Signature =
			append([]byte(nil), signature...)

		tx.Inputs[i].PubKey =
			append([]byte(nil), pub...)
	}

	id, err := CanonicalTransactionIDV1(tx)
	if err != nil {
		return fmt.Errorf(
			"compute canonical transaction id: %w",
			err,
		)
	}
	tx.ID = id

	return nil
}

// Verify documents the QuantumCoin component used by this module.
// İmzaları doğrula (coinbase hariç).
func (tx *Transaction) Verify() bool {
	if tx == nil {
		return false
	}

	if err := ValidateTransactionIDV1(tx); err != nil {
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
		input := tx.Inputs[i]

		if len(input.Signature) == 0 ||
			len(input.PubKey) != 65 ||
			input.PubKey[0] != 0x04 {
			return false
		}

		if !input.UsesKey(fromPKH) {
			return false
		}

		message := signMessageBytes(tx, i)

		var digest [32]byte
		if len(message) != len(digest) {
			return false
		}
		copy(digest[:], message)

		if err := qconsensus.VerifyTransactionDigestSignatureV1(
			input.PubKey,
			digest,
			input.Signature,
		); err != nil {
			return false
		}
	}

	return true
}

// SigningHashes documents the QuantumCoin component used by this module.
// ---- Web cüzdan için: her input’un imzalanacak mesajının HEX’i ----
func SigningHashes(tx *Transaction) []string {
	out := make([]string, 0, len(tx.Inputs))
	for i := range tx.Inputs {
		h := signMessageBytes(tx, i)
		out = append(out, hex.EncodeToString(h))
	}
	return out
}
