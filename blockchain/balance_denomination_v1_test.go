package blockchain

import "testing"

func TestBalanceSnapshotsPreserveLegacyWholeQCV1(
	t *testing.T,
) {
	bc, owner := atomSpendSelectionTestChainV1(
		t,
		100,
	)

	address := owner.GetAddress()

	if got := bc.BalanceSnapshot(address); got != 1 {
		t.Fatalf(
			"legacy total balance=%d want=1 whole QC",
			got,
		)
	}

	if got := bc.SpendableBalanceSnapshot(address); got != 1 {
		t.Fatalf(
			"legacy spendable balance=%d want=1 whole QC",
			got,
		)
	}
}

func TestBalanceSnapshotsNormalizeLegacyUTXOAtAtomActivationV1(
	t *testing.T,
) {
	bc, owner := atomSpendSelectionTestChainV1(
		t,
		1,
	)

	address := owner.GetAddress()

	if got := bc.BalanceSnapshot(address); got != 1_000_000 {
		t.Fatalf(
			"AtomV3 total balance=%d want=1000000 atoms",
			got,
		)
	}

	if got := bc.SpendableBalanceSnapshot(address); got != 1_000_000 {
		t.Fatalf(
			"AtomV3 spendable balance=%d want=1000000 atoms",
			got,
		)
	}
}
