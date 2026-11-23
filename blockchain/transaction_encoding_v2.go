package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
)

// TransactionContentV2 is the pre-launch network-v4 consensus encoding.
// It never uses Gob type identifiers, map order, time zones or process state.
// Historical V1 hashing/signing remains unchanged for old-chain inspection.
func TransactionContentV2(tx *Transaction) ([]byte, error) {
	if tx == nil {
		return nil, ErrNilTransaction
	}
	if len(tx.Inputs) > 5000 || len(tx.Outputs) > 5000 || math.IsNaN(tx.Amount) || math.IsInf(tx.Amount, 0) {
		return nil, fmt.Errorf("invalid canonical transaction content")
	}
	var b bytes.Buffer
	b.WriteString("QC_TX_CONTENT_V2\x00")
	u32 := func(n uint32) { var raw [4]byte; binary.BigEndian.PutUint32(raw[:], n); b.Write(raw[:]) }
	u64 := func(n uint64) { var raw [8]byte; binary.BigEndian.PutUint64(raw[:], n); b.Write(raw[:]) }
	field := func(value []byte) error {
		if len(value) > 1<<20 || b.Len()+4+len(value) > 16<<20 {
			return fmt.Errorf("canonical transaction field too large")
		}
		u32(uint32(len(value)))
		b.Write(value)
		return nil
	}
	u32(uint32(len(tx.Inputs)))
	for _, input := range tx.Inputs {
		if err := field(input.TxID); err != nil {
			return nil, err
		}
		u64(uint64(int64(input.OutIndex)))
	}
	u32(uint32(len(tx.Outputs)))
	for _, output := range tx.Outputs {
		u64(uint64(int64(output.Amount)))
		if err := field(output.PubKeyHash); err != nil {
			return nil, err
		}
	}
	u64(uint64(tx.Timestamp.Unix()))
	u32(uint32(tx.Timestamp.Nanosecond()))
	if err := field([]byte(tx.Sender)); err != nil {
		return nil, err
	}
	u64(math.Float64bits(tx.Amount))
	if b.Len() > 16<<20 {
		return nil, fmt.Errorf("canonical transaction content too large")
	}
	return b.Bytes(), nil
}

// CanonicalTransactionIDV2 retains 32-byte IDs and existing transaction wire
// fields. Only the activated chain's commitment encoding changes.
func CanonicalTransactionIDV2(tx *Transaction) ([]byte, error) {
	raw, err := TransactionContentV2(tx)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(raw)
	return append([]byte(nil), hash[:]...), nil
}

func ValidateTransactionIDV2(tx *Transaction) error {
	if tx == nil || len(tx.ID) != 32 {
		return fmt.Errorf("invalid canonical V2 transaction ID")
	}
	expected, err := CanonicalTransactionIDV2(tx)
	if err != nil {
		return err
	}
	if !bytes.Equal(expected, tx.ID) {
		return fmt.Errorf("transaction ID differs from canonical V2 content")
	}
	return nil
}

// Structural admission recognizes the supported content commitments. Only
// the trusted chain context selects which one is acceptable for consensus.
func validateTransactionContentID(tx *Transaction) error {
	if err := ValidateTransactionIDV2(tx); err == nil {
		return nil
	}
	return ValidateTransactionIDV1(tx)
}

func (bc *Blockchain) assignTransactionIDLockedV2(tx *Transaction) error {
	genesis, err := bc.transactionGenesisLockedV2()
	if err != nil {
		return err
	}
	if genesis != nil {
		tx.ID, err = CanonicalTransactionIDV2(tx)
	} else {
		tx.ID, err = CanonicalTransactionIDV1(tx)
	}
	return err
}

func (bc *Blockchain) validateTransactionIDLockedV2(tx *Transaction) error {
	genesis, err := bc.transactionGenesisLockedV2()
	if err != nil {
		return err
	}
	if genesis != nil {
		return ValidateTransactionIDV2(tx)
	}
	return ValidateTransactionIDV1(tx)
}

func transactionInputDigestV2(tx *Transaction, index int) ([32]byte, error) {
	var zero [32]byte
	if tx == nil || index < 0 || index >= len(tx.Inputs) {
		return zero, fmt.Errorf("invalid canonical signing input")
	}
	raw, err := TransactionContentV2(tx)
	if err != nil {
		return zero, err
	}
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], uint32(index))
	message := append([]byte("QC_TX_INPUT_SIG_V2\x00"), raw...)
	message = append(message, encoded[:]...)
	return sha256.Sum256(message), nil
}
