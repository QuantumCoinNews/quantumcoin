package blockchain

import (
	"testing"

	"quantumcoin/wallet"
)

func TestExactQCTransactionBuilderLegacyOneQCV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			100,
		)

	receiver := wallet.NewWallet()

	tx, err := NewTransactionFromQCAmountV1(
		owner.GetAddress(),
		receiver.GetAddress(),
		"1",
		bc,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(tx.Outputs) != 1 {
		t.Fatalf(
			"outputs=%d want=1",
			len(tx.Outputs),
		)
	}

	if tx.Outputs[0].Amount != 1 {
		t.Fatalf(
			"amount=%d want=1",
			tx.Outputs[0].Amount,
		)
	}
}

func TestExactQCTransactionBuilderAtomOneAtomV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	receiver := wallet.NewWallet()

	tx, err := NewTransactionFromQCAmountV1(
		owner.GetAddress(),
		receiver.GetAddress(),
		"0.000001",
		bc,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(tx.Outputs) != 2 {
		t.Fatalf(
			"outputs=%d want=2",
			len(tx.Outputs),
		)
	}

	if tx.Outputs[0].Amount != 1 {
		t.Fatalf(
			"recipient=%d want=1 atom",
			tx.Outputs[0].Amount,
		)
	}

	if tx.Outputs[1].Amount != 999_999 {
		t.Fatalf(
			"change=%d want=999999 atoms",
			tx.Outputs[1].Amount,
		)
	}
}

func TestExactQCTransactionBuilderAtomTenAtomsV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	receiver := wallet.NewWallet()

	tx, err := NewTransactionFromQCAmountV1(
		owner.GetAddress(),
		receiver.GetAddress(),
		"0.000010",
		bc,
	)
	if err != nil {
		t.Fatal(err)
	}

	if tx.Outputs[0].Amount != 10 {
		t.Fatalf(
			"recipient=%d want=10 atoms",
			tx.Outputs[0].Amount,
		)
	}

	if tx.Outputs[1].Amount != 999_990 {
		t.Fatalf(
			"change=%d want=999990 atoms",
			tx.Outputs[1].Amount,
		)
	}
}

func TestExactQCTransactionBuilderLegacyRejectsFractionV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			100,
		)

	receiver := wallet.NewWallet()

	if _, err := NewTransactionFromQCAmountV1(
		owner.GetAddress(),
		receiver.GetAddress(),
		"0.000001",
		bc,
	); err == nil {
		t.Fatal(
			"LegacyV1 fractional QC unexpectedly accepted",
		)
	}
}
