package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"math/big"

	qconsensus "quantumcoin/consensus"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

const TransactionConsensusVersionV1 = "QC_TRANSACTION_ID_SIGNATURE_V1"

func canonicalTransactionIDPayloadV1(
	tx *Transaction,
) ([]byte, error) {
	if tx == nil {
		return nil, fmt.Errorf(
			"%s: nil transaction",
			TransactionConsensusVersionV1,
		)
	}

	copyTx := tx.TrimmedCopy()
	copyTx.ID = nil

	var buffer bytes.Buffer
	if err := gob.NewEncoder(&buffer).Encode(&copyTx); err != nil {
		return nil, fmt.Errorf(
			"%s: encode transaction id payload: %w",
			TransactionConsensusVersionV1,
			err,
		)
	}

	return buffer.Bytes(), nil
}

// CanonicalTransactionIDV1 returns a stable transaction ID. Input signatures
// and public keys are deliberately excluded, preserving the ID assigned to an
// unsigned transaction after it is signed while binding all spend outpoints,
// outputs, timestamp and transaction convenience fields.
func CanonicalTransactionIDV1(
	tx *Transaction,
) ([]byte, error) {
	payload, err := canonicalTransactionIDPayloadV1(tx)
	if err != nil {
		return nil, err
	}

	transactionID, err :=
		qconsensus.CanonicalTransactionIDFromPayloadV1(
			payload,
		)
	if err != nil {
		return nil, err
	}

	return append(
		[]byte(nil),
		transactionID[:]...,
	), nil
}

func ValidateTransactionIDV1(
	tx *Transaction,
) error {
	if tx == nil {
		return fmt.Errorf(
			"%s: nil transaction",
			TransactionConsensusVersionV1,
		)
	}

	if len(tx.ID) != sha256.Size {
		return fmt.Errorf(
			"%s: transaction id length is %d, want %d",
			TransactionConsensusVersionV1,
			len(tx.ID),
			sha256.Size,
		)
	}

	expected, err := CanonicalTransactionIDV1(tx)
	if err != nil {
		return err
	}

	if !bytes.Equal(tx.ID, expected) {
		return fmt.Errorf(
			"%s: transaction id does not match canonical content",
			TransactionConsensusVersionV1,
		)
	}

	return nil
}

func transactionCurveOrderV1() *big.Int {
	return new(big.Int).Set(
		secp256k1.S256().Params().N,
	)
}

func transactionHalfCurveOrderV1() *big.Int {
	return new(big.Int).Rsh(
		transactionCurveOrderV1(),
		1,
	)
}

func normalizeTransactionLowSV1(
	s *big.Int,
) *big.Int {
	return qconsensus.NormalizeTransactionLowSV1(s)
}

func isMinimalPositiveIntegerEncodingV1(
	encoded []byte,
) bool {
	if len(encoded) == 0 {
		return false
	}

	if len(encoded) > 1 && encoded[0] == 0 {
		return false
	}

	return true
}

func decodeCanonicalTransactionSignatureV1(
	signature []byte,
) (*big.Int, *big.Int, error) {
	return qconsensus.DecodeCanonicalTransactionSignatureV1(
		signature,
	)
}
