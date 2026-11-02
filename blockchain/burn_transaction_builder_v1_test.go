package blockchain

import (
	"bytes"
	"testing"

	"quantumcoin/wallet"
)

func TestCanonicalBurnTransactionBuilderAtomV3V1(t *testing.T) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	tx, err := NewCanonicalBurnTransactionFromQCAmountV1(
		owner.GetAddress(),
		"0.000001",
		bc,
	)
	if err != nil {
		t.Fatalf("create canonical burn transaction: %v", err)
	}

	if tx == nil {
		t.Fatal("canonical burn transaction is nil")
	}

	if len(tx.Inputs) == 0 {
		t.Fatal("canonical burn transaction has no spendable inputs")
	}

	if len(tx.Outputs) != 2 {
		t.Fatalf(
			"burn outputs=%d want=2 (burn + change)",
			len(tx.Outputs),
		)
	}

	if !IsCanonicalBurnOutputV1(tx.Outputs[0]) {
		t.Fatal("first output is not canonical burn output")
	}

	if tx.Outputs[0].Amount != 1 {
		t.Fatalf(
			"burn amount=%d want=1 atom",
			tx.Outputs[0].Amount,
		)
	}

	if IsCanonicalBurnOutputV1(tx.Outputs[1]) {
		t.Fatal("change output unexpectedly classified as burn")
	}

	wantChangePKH :=
		wallet.Base58DecodeAddress(owner.GetAddress())

	if !bytes.Equal(
		tx.Outputs[1].PubKeyHash,
		wantChangePKH,
	) {
		t.Fatal("change output does not return to owner")
	}

	if tx.Outputs[1].Amount != 999_999 {
		t.Fatalf(
			"change=%d want=999999 atoms",
			tx.Outputs[1].Amount,
		)
	}

	if err := tx.Sign(owner.PrivateKey); err != nil {
		t.Fatalf("sign canonical burn transaction: %v", err)
	}

	if !tx.Verify() {
		t.Fatal("signed canonical burn transaction did not verify")
	}
}

func TestCanonicalBurnTransactionRejectsWrongOwnerKeyV1(t *testing.T) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	tx, err := NewCanonicalBurnTransactionFromQCAmountV1(
		owner.GetAddress(),
		"0.000001",
		bc,
	)
	if err != nil {
		t.Fatalf("create canonical burn transaction: %v", err)
	}

	attacker := wallet.NewWallet()

	if err := tx.Sign(attacker.PrivateKey); err != nil {
		t.Fatalf(
			"sign with wrong owner key unexpectedly failed before verification: %v",
			err,
		)
	}

	if tx.Verify() {
		t.Fatal("canonical burn transaction signed by wrong owner unexpectedly verified")
	}
}
