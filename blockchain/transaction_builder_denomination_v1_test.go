package blockchain

import "testing"

func TestTransactionBuilderWholeQCLegacyRemainsWholeQCV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 100)

	got, err :=
		bc.wholeQCAmountToTransactionUnitsAtHeightV1(
			1,
			99,
		)
	if err != nil {
		t.Fatal(err)
	}

	if got != 1 {
		t.Fatalf("got=%d want=1", got)
	}
}

func TestTransactionBuilderWholeQCBecomesAtomsAtAtomV3V1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 1)

	got, err :=
		bc.wholeQCAmountToTransactionUnitsAtHeightV1(
			1,
			1,
		)
	if err != nil {
		t.Fatal(err)
	}

	if got != 1_000_000 {
		t.Fatalf(
			"got=%d want=1000000 atoms",
			got,
		)
	}
}

func TestTransactionBuilderFiftyQCBecomesFiftyMillionAtomsV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 1)

	got, err :=
		bc.wholeQCAmountToTransactionUnitsAtHeightV1(
			50,
			1,
		)
	if err != nil {
		t.Fatal(err)
	}

	if got != 50_000_000 {
		t.Fatalf(
			"got=%d want=50000000 atoms",
			got,
		)
	}
}

func TestTransactionBuilderRejectsNonPositiveWholeQCV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 1)

	for _, amount := range []int{0, -1} {
		if _, err :=
			bc.wholeQCAmountToTransactionUnitsAtHeightV1(
				amount,
				1,
			); err == nil {
			t.Fatalf(
				"amount=%d unexpectedly accepted",
				amount,
			)
		}
	}
}
