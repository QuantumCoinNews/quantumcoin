package blockchain

import (
	"bytes"
	"testing"

	"quantumcoin/wallet"
)

func TestResolveCanonicalBurnOwnerFromInputUTXOV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	tx, err :=
		NewCanonicalBurnTransactionFromQCAmountV1(
			owner.GetAddress(),
			"0.000001",
			bc,
		)
	if err != nil {
		t.Fatalf(
			"create canonical burn transaction: %v",
			err,
		)
	}

	if err := tx.Sign(owner.PrivateKey); err != nil {
		t.Fatalf(
			"sign canonical burn transaction: %v",
			err,
		)
	}

	ownerKey, err :=
		bc.ResolveCanonicalBurnOwnerKeyV1(
			tx,
		)
	if err != nil {
		t.Fatalf(
			"resolve canonical burn owner: %v",
			err,
		)
	}

	want :=
		wallet.Base58DecodeAddress(
			owner.GetAddress(),
		)

	if len(want) == 0 {
		t.Fatal(
			"owner address decoded to empty key",
		)
	}

	if !bytes.Equal(ownerKey, want) {
		t.Fatalf(
			"resolved owner key=%x want=%x",
			ownerKey,
			want,
		)
	}
}

func TestResolveCanonicalBurnOwnerIgnoresSenderTextV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	tx, err :=
		NewCanonicalBurnTransactionFromQCAmountV1(
			owner.GetAddress(),
			"0.000001",
			bc,
		)
	if err != nil {
		t.Fatalf(
			"create canonical burn transaction: %v",
			err,
		)
	}

	if err := tx.Sign(owner.PrivateKey); err != nil {
		t.Fatalf(
			"sign canonical burn transaction: %v",
			err,
		)
	}

	want :=
		wallet.Base58DecodeAddress(
			owner.GetAddress(),
		)

	tx.Sender = "UNTRUSTED_ACCOUNTING_LABEL"

	ownerKey, err :=
		bc.ResolveCanonicalBurnOwnerKeyV1(
			tx,
		)
	if err != nil {
		t.Fatalf(
			"resolve owner independently of sender text: %v",
			err,
		)
	}

	if !bytes.Equal(ownerKey, want) {
		t.Fatalf(
			"sender text changed resolved owner key: got=%x want=%x",
			ownerKey,
			want,
		)
	}
}
