package blockchain

import "testing"

func TestClassifyCanonicalBurnTransactionSumsAtomV3OutputsV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(
		t,
		1,
	)

	tx := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     []byte{0x01},
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			NewCanonicalBurnOutputV1(1),
			NewCanonicalBurnOutputV1(9),
		},
	}

	result, err :=
		ClassifyCanonicalBurnTransactionAtHeightV1(
			tx,
			1,
			bc,
		)
	if err != nil {
		t.Fatalf(
			"classify AtomV3 multi-output burn: %v",
			err,
		)
	}

	if !result.IsBurn {
		t.Fatal(
			"AtomV3 multi-output burn was not classified as burn",
		)
	}

	if result.BurnAmountAtoms != 10 {
		t.Fatalf(
			"AtomV3 burn atoms=%d want=10",
			result.BurnAmountAtoms,
		)
	}
}

func TestClassifyCanonicalBurnTransactionSumsLegacyOutputsAsAtomsV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(
		t,
		100,
	)

	tx := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     []byte{0x01},
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			NewCanonicalBurnOutputV1(1),
			NewCanonicalBurnOutputV1(9),
		},
	}

	result, err :=
		ClassifyCanonicalBurnTransactionAtHeightV1(
			tx,
			1,
			bc,
		)
	if err != nil {
		t.Fatalf(
			"classify LegacyV1 multi-output burn: %v",
			err,
		)
	}

	if !result.IsBurn {
		t.Fatal(
			"LegacyV1 multi-output burn was not classified as burn",
		)
	}

	const wantAtoms int64 = 10_000_000

	if result.BurnAmountAtoms != wantAtoms {
		t.Fatalf(
			"LegacyV1 burn atoms=%d want=%d",
			result.BurnAmountAtoms,
			wantAtoms,
		)
	}
}
