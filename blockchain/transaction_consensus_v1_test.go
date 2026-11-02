package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"quantumcoin/wallet"
)

func legacyUnsignedTransactionHashForTestV1(
	tx *Transaction,
) []byte {
	copyTx := *tx
	copyTx.ID = nil
	sum := sha256.Sum256(copyTx.Serialize())
	return append([]byte(nil), sum[:]...)
}

func testCanonicalSignedTransactionV1(
	t *testing.T,
) *Transaction {
	t.Helper()

	sender := wallet.NewWallet()
	receiver := wallet.NewWallet()

	tx := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     bytes.Repeat([]byte{0x42}, 32),
				OutIndex: 3,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount: 7,
				PubKeyHash: wallet.HashPubKey(
					receiver.PublicKey,
				),
			},
		},
		Timestamp: time.Unix(1_725_000_123, 0).UTC(),
		Sender:    sender.GetAddress(),
		Amount:    7,
	}
	tx.ID = tx.Hash()

	if err := tx.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("sign transaction: %v", err)
	}
	if !tx.Verify() {
		t.Fatalf("signed canonical transaction did not verify")
	}

	return tx
}

func rawSignatureEncodingForTestV1(
	r *big.Int,
	s *big.Int,
) []byte {
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	signature := make(
		[]byte,
		0,
		2+len(rBytes)+len(sBytes),
	)
	signature = append(signature, byte(len(rBytes)))
	signature = append(signature, rBytes...)
	signature = append(signature, byte(len(sBytes)))
	signature = append(signature, sBytes...)

	return signature
}

func TestTransactionIDStableAcrossSigningV1(
	t *testing.T,
) {
	sender := wallet.NewWallet()
	receiver := wallet.NewWallet()

	tx := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     bytes.Repeat([]byte{0x11}, 32),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount: 5,
				PubKeyHash: wallet.HashPubKey(
					receiver.PublicKey,
				),
			},
		},
		Timestamp: time.Unix(1_725_000_456, 0).UTC(),
		Sender:    sender.GetAddress(),
		Amount:    5,
	}
	legacyUnsignedID := legacyUnsignedTransactionHashForTestV1(tx)
	tx.ID = tx.Hash()
	unsignedID := append([]byte(nil), tx.ID...)

	if !bytes.Equal(legacyUnsignedID, unsignedID) {
		t.Fatalf(
			"canonical transaction id changed unsigned legacy semantics",
		)
	}

	if err := tx.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("sign transaction: %v", err)
	}

	if !bytes.Equal(unsignedID, tx.ID) {
		t.Fatalf(
			"transaction id changed after adding signatures",
		)
	}
	if err := ValidateTransactionIDV1(tx); err != nil {
		t.Fatalf("canonical transaction id rejected: %v", err)
	}
}

func TestTransactionIDMutationRejectedV1(
	t *testing.T,
) {
	tx := testCanonicalSignedTransactionV1(t)
	tx.Outputs[0].Amount++

	if err := ValidateTransactionIDV1(tx); err == nil {
		t.Fatalf("mutated transaction content kept a valid id")
	}
	if tx.Verify() {
		t.Fatalf("mutated transaction verified")
	}
}

func TestTransactionBasicValidationRejectsWrongIDV1(
	t *testing.T,
) {
	tx := testCanonicalSignedTransactionV1(t)
	tx.ID[0] ^= 0xff

	if err := ValidateTransactionBasic(tx); err == nil {
		t.Fatalf("basic validation accepted wrong transaction id")
	}
}

func TestTransactionSignProducesLowSV1(
	t *testing.T,
) {
	tx := testCanonicalSignedTransactionV1(t)

	_, s, err := decodeCanonicalTransactionSignatureV1(
		tx.Inputs[0].Signature,
	)
	if err != nil {
		t.Fatalf("decode canonical signature: %v", err)
	}
	if s.Cmp(transactionHalfCurveOrderV1()) > 0 {
		t.Fatalf("Sign produced a high-S signature")
	}
}

func TestTransactionRejectsHighSMalleabilityV1(
	t *testing.T,
) {
	tx := testCanonicalSignedTransactionV1(t)

	r, s, err := decodeCanonicalTransactionSignatureV1(
		tx.Inputs[0].Signature,
	)
	if err != nil {
		t.Fatalf("decode canonical signature: %v", err)
	}

	highS := new(big.Int).Sub(
		transactionCurveOrderV1(),
		s,
	)
	if highS.Cmp(transactionHalfCurveOrderV1()) <= 0 {
		t.Fatalf("test high-S fixture is not high")
	}

	tx.Inputs[0].Signature =
		rawSignatureEncodingForTestV1(r, highS)

	if tx.Verify() {
		t.Fatalf("high-S malleated signature verified")
	}
}

func TestTransactionRejectsNonMinimalSignatureV1(
	t *testing.T,
) {
	tx := testCanonicalSignedTransactionV1(t)

	r, s, err := decodeCanonicalTransactionSignatureV1(
		tx.Inputs[0].Signature,
	)
	if err != nil {
		t.Fatalf("decode canonical signature: %v", err)
	}

	rBytes := append([]byte{0}, r.Bytes()...)
	sBytes := s.Bytes()

	signature := []byte{byte(len(rBytes))}
	signature = append(signature, rBytes...)
	signature = append(signature, byte(len(sBytes)))
	signature = append(signature, sBytes...)
	tx.Inputs[0].Signature = signature

	if tx.Verify() {
		t.Fatalf("non-minimal signature verified")
	}
}

func TestTransactionRejectsTrailingSignatureDataV1(
	t *testing.T,
) {
	tx := testCanonicalSignedTransactionV1(t)
	tx.Inputs[0].Signature = append(
		tx.Inputs[0].Signature,
		0x00,
	)

	if tx.Verify() {
		t.Fatalf("signature with trailing data verified")
	}
}

func TestCoinbaseRequiresCanonicalTransactionIDV1(
	t *testing.T,
) {
	tx := &Transaction{
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     50,
				PubKeyHash: []byte("miner"),
			},
		},
		Timestamp: time.Unix(1_725_000_789, 0).UTC(),
		Sender:    "COINBASE",
		Amount:    50,
	}
	tx.ID = tx.Hash()

	if !tx.Verify() {
		t.Fatalf("canonical coinbase did not verify")
	}

	tx.ID[0] ^= 0x01
	if tx.Verify() {
		t.Fatalf("coinbase with wrong id verified")
	}
}
func legacySigningDigestForTestV1(
	tx *Transaction,
	inputIndex int,
) []byte {
	txCopy := tx.TrimmedCopy()
	txCopy.ID = nil

	var payload bytes.Buffer
	payload.Write(txCopy.Serialize())

	input := tx.Inputs[inputIndex]
	payload.Write(input.TxID)

	var encodedOutputIndex [4]byte
	binary.BigEndian.PutUint32(
		encodedOutputIndex[:],
		uint32(input.OutIndex),
	)
	payload.Write(encodedOutputIndex[:])

	sum := sha256.Sum256(payload.Bytes())
	return append([]byte(nil), sum[:]...)
}

func TestSigningDigestPreservesLegacyBytesV1(
	t *testing.T,
) {
	testCases := []struct {
		name     string
		txID     []byte
		outIndex int
	}{
		{
			name: "normal outpoint",
			txID: bytes.Repeat(
				[]byte{0x5a},
				sha256.Size,
			),
			outIndex: 3,
		},
		{
			name:     "legacy empty transaction id",
			txID:     nil,
			outIndex: 3,
		},
		{
			name: "legacy uint32 index conversion",
			txID: bytes.Repeat(
				[]byte{0x6b},
				sha256.Size,
			),
			outIndex: -1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tx := &Transaction{
				Inputs: []TransactionInput{
					{
						TxID: append(
							[]byte(nil),
							testCase.txID...,
						),
						OutIndex: testCase.outIndex,
					},
				},
				Outputs: []TransactionOutput{
					{
						Amount:     7,
						PubKeyHash: []byte("receiver"),
					},
				},
				Timestamp: time.Unix(
					1_725_001_234,
					0,
				).UTC(),
				Sender: "legacy-signing-test",
				Amount: 7,
			}

			expected :=
				legacySigningDigestForTestV1(tx, 0)

			actual := signMessageBytes(tx, 0)

			if !bytes.Equal(actual, expected) {
				t.Fatalf(
					"signing digest changed legacy bytes:\nactual=%x\nexpected=%x",
					actual,
					expected,
				)
			}
		})
	}
}
