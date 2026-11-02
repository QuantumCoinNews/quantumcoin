package blockchain

import (
	"testing"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

func historicalBurnForOwnerV1(
	t *testing.T,
	owner *wallet.Wallet,
	amount int,
	tag byte,
) *Transaction {
	t.Helper()

	if owner == nil {
		t.Fatal("nil owner wallet")
	}

	return &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     []byte{tag},
				OutIndex: 0,
				PubKey: append(
					[]byte(nil),
					owner.PublicKey...,
				),
			},
		},
		Outputs: []TransactionOutput{
			NewCanonicalBurnOutputV1(amount),
		},
	}
}

func TestTotalBurnedAtomsByOwnerCanonicalReorgV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	ownerKey :=
		wallet.HashPubKey(
			owner.PublicKey,
		)

	originalBlocks :=
		append(
			[]*Block(nil),
			bc.Blocks...,
		)

	oldBurn :=
		historicalBurnForOwnerV1(
			t,
			owner,
			3,
			0x31,
		)

	bc.Blocks = append(
		append(
			[]*Block(nil),
			originalBlocks...,
		),
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				oldBurn,
			},
		},
	)

	got, err :=
		bc.TotalBurnedAtomsByOwnerCheckedV1(
			ownerKey,
		)
	if err != nil {
		t.Fatalf(
			"old canonical accounting: %v",
			err,
		)
	}

	if got != 3 {
		t.Fatalf(
			"old canonical burned=%d want=3 atoms",
			got,
		)
	}

	newBurn :=
		historicalBurnForOwnerV1(
			t,
			owner,
			7,
			0x32,
		)

	// Simulate a canonical reorg by replacing the old height-1 block.
	// The orphaned 3-atom burn must disappear from lifetime accounting.
	bc.Blocks = append(
		append(
			[]*Block(nil),
			originalBlocks...,
		),
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				newBurn,
			},
		},
	)

	got, err =
		bc.TotalBurnedAtomsByOwnerCheckedV1(
			ownerKey,
		)
	if err != nil {
		t.Fatalf(
			"reorg canonical accounting: %v",
			err,
		)
	}

	if got != 7 {
		t.Fatalf(
			"reorg canonical burned=%d want=7 atoms",
			got,
		)
	}
}

func TestTotalBurnedAtomsByOwnerLegacyToAtomV3V1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			2,
		)

	ownerKey :=
		wallet.HashPubKey(
			owner.PublicKey,
		)

	legacyBurn :=
		historicalBurnForOwnerV1(
			t,
			owner,
			1,
			0x41,
		)

	atomBurn :=
		historicalBurnForOwnerV1(
			t,
			owner,
			1,
			0x42,
		)

	bc.Blocks = append(
		bc.Blocks,
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				legacyBurn,
			},
		},
		&Block{
			Index: 2,
			Transactions: []*Transaction{
				atomBurn,
			},
		},
	)

	got, err :=
		bc.TotalBurnedAtomsByOwnerCheckedV1(
			ownerKey,
		)
	if err != nil {
		t.Fatalf(
			"legacy-to-atom accounting: %v",
			err,
		)
	}

	want :=
		qconsensus.CanonicalAtomsPerQCV1 + 1

	if got != want {
		t.Fatalf(
			"legacy-to-atom burned=%d want=%d atoms",
			got,
			want,
		)
	}
}
