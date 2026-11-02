package blockchain

import (
	"testing"

	"quantumcoin/wallet"
)

func TestTotalBurnedAtomsByOwnerCheckedV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	other := wallet.NewWallet()
	if other == nil {
		t.Fatal(
			"create other wallet returned nil",
		)
	}

	ownerKey :=
		wallet.HashPubKey(
			owner.PublicKey,
		)

	otherKey :=
		wallet.HashPubKey(
			other.PublicKey,
		)

	ownerBurnOne := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     []byte{0x01},
				OutIndex: 0,
				PubKey: append(
					[]byte(nil),
					owner.PublicKey...,
				),
			},
		},
		Outputs: []TransactionOutput{
			NewCanonicalBurnOutputV1(
				3,
			),
		},
	}

	ownerBurnTwo := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     []byte{0x02},
				OutIndex: 0,
				PubKey: append(
					[]byte(nil),
					owner.PublicKey...,
				),
			},
		},
		Outputs: []TransactionOutput{
			NewCanonicalBurnOutputV1(
				7,
			),
		},
	}

	otherBurn := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     []byte{0x03},
				OutIndex: 0,
				PubKey: append(
					[]byte(nil),
					other.PublicKey...,
				),
			},
		},
		Outputs: []TransactionOutput{
			NewCanonicalBurnOutputV1(
				11,
			),
		},
	}

	bc.Blocks = append(
		bc.Blocks,
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				ownerBurnOne,
				otherBurn,
				ownerBurnTwo,
			},
		},
	)

	gotOwner, err :=
		bc.TotalBurnedAtomsByOwnerCheckedV1(
			ownerKey,
		)
	if err != nil {
		t.Fatalf(
			"owner lifetime burn accounting: %v",
			err,
		)
	}

	if gotOwner != 10 {
		t.Fatalf(
			"owner lifetime burned=%d want=10 atoms",
			gotOwner,
		)
	}

	gotOther, err :=
		bc.TotalBurnedAtomsByOwnerCheckedV1(
			otherKey,
		)
	if err != nil {
		t.Fatalf(
			"other owner lifetime burn accounting: %v",
			err,
		)
	}

	if gotOther != 11 {
		t.Fatalf(
			"other owner lifetime burned=%d want=11 atoms",
			gotOther,
		)
	}
}

func TestTotalBurnedAtomsByOwnerCheckedRejectsEmptyOwnerV1(
	t *testing.T,
) {
	bc, _ :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	if _, err :=
		bc.TotalBurnedAtomsByOwnerCheckedV1(
			nil,
		); err == nil {
		t.Fatal(
			"empty owner key unexpectedly accepted",
		)
	}
}
