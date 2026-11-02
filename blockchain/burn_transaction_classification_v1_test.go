package blockchain

import "testing"

func TestClassifyCanonicalBurnTransactionAtomV3V1(
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

	result, err :=
		ClassifyCanonicalBurnTransactionAtHeightV1(
			tx,
			1,
			bc,
		)
	if err != nil {
		t.Fatalf(
			"classify canonical burn transaction: %v",
			err,
		)
	}

	if !result.IsBurn {
		t.Fatal(
			"canonical burn transaction was not classified as burn",
		)
	}

	if result.BurnAmountAtoms != 1 {
		t.Fatalf(
			"burn amount atoms=%d want=1",
			result.BurnAmountAtoms,
		)
	}
}

func TestClassifyNormalTransactionIsNotBurnV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	receiver := NewCanonicalBurnOutputV1(1)
	receiver.PubKeyHash = []byte{
		0x01, 0x02, 0x03, 0x04,
	}

	tx := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID: append(
					[]byte(nil),
					bc.Blocks[0].Transactions[0].ID...,
				),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			receiver,
		},
		Sender: owner.GetAddress(),
	}

	tx.ID = tx.Hash()

	result, err :=
		ClassifyCanonicalBurnTransactionAtHeightV1(
			tx,
			1,
			bc,
		)
	if err != nil {
		t.Fatalf(
			"classify normal transaction: %v",
			err,
		)
	}

	if result.IsBurn {
		t.Fatal(
			"normal transaction unexpectedly classified as burn",
		)
	}

	if result.BurnAmountAtoms != 0 {
		t.Fatalf(
			"normal burn amount atoms=%d want=0",
			result.BurnAmountAtoms,
		)
	}
}
