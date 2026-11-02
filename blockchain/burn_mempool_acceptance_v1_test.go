package blockchain

import (
	"strings"
	"testing"

	"quantumcoin/wallet"
)

func TestCanonicalBurnMempoolAcceptsOwnerSignedAtomV3V1(
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

	if !tx.Verify() {
		t.Fatal(
			"owner-signed canonical burn transaction did not verify",
		)
	}

	if err := bc.AddTransaction(tx); err != nil {
		t.Fatalf(
			"owner-signed canonical burn rejected by mempool: %v",
			err,
		)
	}

	if len(bc.pendingTxs) != 1 {
		t.Fatalf(
			"pending tx count=%d want=1",
			len(bc.pendingTxs),
		)
	}

	if bc.pendingTxs[0] != tx {
		t.Fatal(
			"accepted canonical burn transaction not found in mempool",
		)
	}
}

func TestCanonicalBurnMempoolRejectsWrongOwnerKeyAtomV3V1(
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

	attacker := wallet.NewWallet()

	if err := tx.Sign(attacker.PrivateKey); err != nil {
		t.Fatalf(
			"sign with wrong owner key before verification: %v",
			err,
		)
	}

	err = bc.AddTransaction(tx)
	if err == nil {
		t.Fatal(
			"wrong-owner canonical burn unexpectedly accepted by mempool",
		)
	}

	if !strings.Contains(
		err.Error(),
		"invalid tx signature",
	) {
		t.Fatalf(
			"wrong-owner rejection=%q want invalid tx signature",
			err.Error(),
		)
	}

	if len(bc.pendingTxs) != 0 {
		t.Fatalf(
			"wrong-owner burn entered mempool; pending=%d want=0",
			len(bc.pendingTxs),
		)
	}
}
