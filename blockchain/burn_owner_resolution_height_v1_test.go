package blockchain

import (
	"bytes"
	"testing"

	"quantumcoin/wallet"
)

func TestResolveCanonicalBurnOwnerAgainstUTXOStateAtHeightV1(
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

	scratch :=
		cloneUTXOSetV1(
			bc.UTXO,
		)

	ownerKey, err :=
		bc.resolveCanonicalBurnOwnerKeyAgainstUTXOStateAtHeightV1(
			tx,
			scratch,
			1,
		)
	if err != nil {
		t.Fatalf(
			"resolve burn owner at explicit height: %v",
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

func TestResolveCanonicalBurnOwnerRejectsNegativeHeightV1(
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

	_, err =
		bc.resolveCanonicalBurnOwnerKeyAgainstUTXOStateAtHeightV1(
			tx,
			cloneUTXOSetV1(bc.UTXO),
			-1,
		)

	if err == nil {
		t.Fatal(
			"negative candidate height unexpectedly accepted",
		)
	}
}
