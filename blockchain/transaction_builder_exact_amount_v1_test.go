package blockchain

import "testing"

func TestExactQCAmountLegacyWholeQCV1(t *testing.T) {
	bc := atomDenominationTestChainV1(t, 100)

	got, err :=
		bc.qcAmountStringToTransactionUnitsAtHeightV1(
			"1",
			99,
		)
	if err != nil {
		t.Fatal(err)
	}

	if got != 1 {
		t.Fatalf("got=%d want=1", got)
	}
}

func TestExactQCAmountLegacyRejectsFractionV1(t *testing.T) {
	bc := atomDenominationTestChainV1(t, 100)

	if _, err :=
		bc.qcAmountStringToTransactionUnitsAtHeightV1(
			"0.000001",
			99,
		); err == nil {
		t.Fatal("LegacyV1 fractional QC unexpectedly accepted")
	}
}

func TestExactQCAmountAtomOneAtomV1(t *testing.T) {
	bc := atomDenominationTestChainV1(t, 1)

	got, err :=
		bc.qcAmountStringToTransactionUnitsAtHeightV1(
			"0.000001",
			1,
		)
	if err != nil {
		t.Fatal(err)
	}

	if got != 1 {
		t.Fatalf("got=%d want=1 atom", got)
	}
}

func TestExactQCAmountAtomTenAtomsV1(t *testing.T) {
	bc := atomDenominationTestChainV1(t, 1)

	got, err :=
		bc.qcAmountStringToTransactionUnitsAtHeightV1(
			"0.000010",
			1,
		)
	if err != nil {
		t.Fatal(err)
	}

	if got != 10 {
		t.Fatalf("got=%d want=10 atoms", got)
	}
}

func TestExactQCAmountAtomOneQCV1(t *testing.T) {
	bc := atomDenominationTestChainV1(t, 1)

	got, err :=
		bc.qcAmountStringToTransactionUnitsAtHeightV1(
			"1",
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

func TestExactQCAmountRejectsInvalidValuesV1(t *testing.T) {
	bc := atomDenominationTestChainV1(t, 1)

	for _, raw := range []string{
		"",
		"0",
		"-1",
		"NaN",
		"0.0000001",
		"1e-6",
	} {
		if _, err :=
			bc.qcAmountStringToTransactionUnitsAtHeightV1(
				raw,
				1,
			); err == nil {
			t.Fatalf(
				"raw=%q unexpectedly accepted",
				raw,
			)
		}
	}
}
