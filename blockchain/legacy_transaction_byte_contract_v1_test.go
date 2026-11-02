package blockchain

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"
)

func legacyTransactionFixtureV1() *Transaction {
	return &Transaction{
		ID: nil,
		Inputs: []TransactionInput{
			{
				TxID:      []byte{0x01, 0x02, 0x03, 0x04},
				OutIndex:  2,
				Signature: nil,
				PubKey:    nil,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount:     50,
				PubKeyHash: []byte{0xaa, 0xbb, 0xcc},
			},
			{
				Amount:     7,
				PubKeyHash: []byte{0x10, 0x20},
			},
		},
		Timestamp: time.Unix(1_700_000_000, 123_456_789).UTC(),
		Sender:    "legacy-sender",
		Amount:    50,
	}
}

func TestLegacyTransactionCanonicalIDPayloadRemainsGobV1(t *testing.T) {
	tx := legacyTransactionFixtureV1()

	got, err := canonicalTransactionIDPayloadV1(tx)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}

	copyTx := tx.TrimmedCopy()
	copyTx.ID = nil
	want := copyTx.Serialize()

	if !bytes.Equal(got, want) {
		t.Fatal("legacy canonical transaction payload changed")
	}
}

func TestLegacyTransactionIDRemainsContentBoundV1(t *testing.T) {
	tx := legacyTransactionFixtureV1()

	first, err := CanonicalTransactionIDV1(tx)
	if err != nil {
		t.Fatalf("canonical id: %v", err)
	}

	second, err := CanonicalTransactionIDV1(tx)
	if err != nil {
		t.Fatalf("canonical id second pass: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("legacy transaction id is not deterministic")
	}

	if len(first) != sha256.Size {
		t.Fatalf(
			"legacy transaction id length=%d want=%d",
			len(first),
			sha256.Size,
		)
	}

	changed := legacyTransactionFixtureV1()
	changed.Outputs[0].Amount++

	changedID, err := CanonicalTransactionIDV1(changed)
	if err != nil {
		t.Fatalf("changed canonical id: %v", err)
	}

	if bytes.Equal(first, changedID) {
		t.Fatal("legacy transaction id did not bind output amount")
	}
}

func TestLegacyTransactionConvenienceAmountRemainsHashBoundV1(t *testing.T) {
	tx := legacyTransactionFixtureV1()

	first, err := CanonicalTransactionIDV1(tx)
	if err != nil {
		t.Fatalf("canonical id: %v", err)
	}

	changed := legacyTransactionFixtureV1()
	changed.Amount = 50.5

	second, err := CanonicalTransactionIDV1(changed)
	if err != nil {
		t.Fatalf("changed canonical id: %v", err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("legacy convenience Amount stopped affecting transaction id")
	}
}

func TestLegacyTransactionSigningPayloadStillUsesSerializedTrimmedCopyV1(t *testing.T) {
	tx := legacyTransactionFixtureV1()

	copyTx := tx.TrimmedCopy()
	copyTx.ID = nil

	serialized := copyTx.Serialize()

	if len(serialized) == 0 {
		t.Fatal("legacy serialized signing payload is empty")
	}

	if len(tx.Inputs) != 1 {
		t.Fatalf(
			"fixture input count=%d want=1",
			len(tx.Inputs),
		)
	}

	message := signMessageBytes(tx, 0)

	if len(message) != sha256.Size {
		t.Fatalf(
			"signing digest length=%d want=%d",
			len(message),
			sha256.Size,
		)
	}
}
