package blockchain

import (
	"testing"

	"quantumcoin/wallet"
)

func TestNewTransactionLegacyCompatibilityUsesWholeQCV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			100,
		)

	receiver := wallet.NewWallet()

	tx, err := NewTransaction(
		owner.GetAddress(),
		receiver.GetAddress(),
		1,
		bc,
	)
	if err != nil {
		t.Fatalf(
			"legacy NewTransaction: %v",
			err,
		)
	}

	if len(tx.Outputs) != 1 {
		t.Fatalf(
			"legacy outputs=%d want=1",
			len(tx.Outputs),
		)
	}

	if tx.Outputs[0].Amount != 1 {
		t.Fatalf(
			"legacy recipient amount=%d want=1",
			tx.Outputs[0].Amount,
		)
	}

	if tx.Amount != 1 {
		t.Fatalf(
			"legacy convenience amount=%v want=1",
			tx.Amount,
		)
	}
}

func TestNewTransactionCompatibilityConvertsWholeQCAtAtomV3V1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	receiver := wallet.NewWallet()

	tx, err := NewTransaction(
		owner.GetAddress(),
		receiver.GetAddress(),
		1,
		bc,
	)
	if err != nil {
		t.Fatalf(
			"AtomV3 compatibility NewTransaction: %v",
			err,
		)
	}

	if len(tx.Outputs) != 1 {
		t.Fatalf(
			"AtomV3 outputs=%d want=1",
			len(tx.Outputs),
		)
	}

	if tx.Outputs[0].Amount != 1_000_000 {
		t.Fatalf(
			"AtomV3 recipient amount=%d want=1000000 atoms",
			tx.Outputs[0].Amount,
		)
	}

	if tx.Amount != 1_000_000 {
		t.Fatalf(
			"AtomV3 convenience amount=%v want=1000000",
			tx.Amount,
		)
	}
}

func TestNewTransactionProtocolUnitsCanBuildOneAtomV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	receiver := wallet.NewWallet()

	tx, err :=
		NewTransactionInProtocolUnitsV1(
			owner.GetAddress(),
			receiver.GetAddress(),
			1,
			bc,
		)
	if err != nil {
		t.Fatalf(
			"one-atom transaction: %v",
			err,
		)
	}

	if len(tx.Outputs) != 2 {
		t.Fatalf(
			"one-atom outputs=%d want=2",
			len(tx.Outputs),
		)
	}

	if tx.Outputs[0].Amount != 1 {
		t.Fatalf(
			"recipient amount=%d want=1 atom",
			tx.Outputs[0].Amount,
		)
	}

	if tx.Outputs[1].Amount != 999_999 {
		t.Fatalf(
			"change amount=%d want=999999 atoms",
			tx.Outputs[1].Amount,
		)
	}

	if tx.Amount != 1 {
		t.Fatalf(
			"convenience amount=%v want=1 protocol unit",
			tx.Amount,
		)
	}
}

func TestNewTransactionProtocolUnitsRejectsNonPositiveV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	receiver := wallet.NewWallet()

	for _, amount := range []int{0, -1} {
		if _, err :=
			NewTransactionInProtocolUnitsV1(
				owner.GetAddress(),
				receiver.GetAddress(),
				amount,
				bc,
			); err == nil {
			t.Fatalf(
				"protocol amount=%d unexpectedly accepted",
				amount,
			)
		}
	}
}
