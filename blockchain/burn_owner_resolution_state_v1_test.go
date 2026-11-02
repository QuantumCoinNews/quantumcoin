package blockchain

import (
	"bytes"
	"encoding/hex"
	"testing"

	"quantumcoin/wallet"
)

func TestResolveCanonicalBurnOwnerAgainstCandidateUTXOStateV1(
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

	// Prove that resolution uses the supplied candidate state rather than
	// silently falling back to bc.UTXO.
	for _, input := range tx.Inputs {
		delete(
			bc.UTXO,
			hex.EncodeToString(input.TxID),
		)
	}

	ownerKey, err :=
		bc.resolveCanonicalBurnOwnerKeyAgainstUTXOStateV1(
			tx,
			scratch,
		)
	if err != nil {
		t.Fatalf(
			"resolve owner against candidate UTXO state: %v",
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
