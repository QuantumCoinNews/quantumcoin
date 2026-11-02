package blockchain

import (
	"bytes"
	"testing"
	"time"

	"quantumcoin/wallet"
)

func TestResolveCanonicalBurnOwnerFromIntraBlockCreatedUTXOV1(
	t *testing.T,
) {
	bc, sender :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)
	middle := wallet.NewWallet()
	if middle == nil {
		t.Fatal(
			"create middle wallet returned nil",
		)
	}

	funding :=
		bc.Blocks[0].Transactions[0]

	if funding == nil {
		t.Fatal(
			"missing funding transaction",
		)
	}

	first := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID: append(
					[]byte(nil),
					funding.ID...,
				),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount: 999_999,
				PubKeyHash: wallet.HashPubKey(
					middle.PublicKey,
				),
			},
		},
		Timestamp: time.Unix(
			CanonicalGenesisUnixV1+60,
			0,
		).UTC(),
		Sender: sender.GetAddress(),
		Amount: 999_999,
	}

	first.ID = first.Hash()

	if err := first.Sign(
		sender.PrivateKey,
	); err != nil {
		t.Fatalf(
			"sign intra-block parent: %v",
			err,
		)
	}

	burn := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID: append(
					[]byte(nil),
					first.ID...,
				),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			NewCanonicalBurnOutputV1(
				999_998,
			),
		},
		Timestamp: time.Unix(
			CanonicalGenesisUnixV1+61,
			0,
		).UTC(),
		Sender: middle.GetAddress(),
		Amount: 999_998,
	}

	burn.ID = burn.Hash()

	if err := burn.Sign(
		middle.PrivateKey,
	); err != nil {
		t.Fatalf(
			"sign intra-block burn: %v",
			err,
		)
	}

	scratch :=
		cloneUTXOSetV1(
			bc.UTXO,
		)

	origins :=
		bc.utxoOriginsV1()

	createdNormalTx :=
		make(map[string]bool)

	spentInSequence :=
		make(map[string]bool)

	knownTxIDs :=
		bc.knownTransactionIDsV1()

	if _, err :=
		bc.validateTransactionAgainstUTXOStateV1(
			first,
			scratch,
			origins,
			createdNormalTx,
			spentInSequence,
			knownTxIDs,
			false,
			1,
		); err != nil {
		t.Fatalf(
			"validate intra-block parent: %v",
			err,
		)
	}

	// The parent is intentionally not committed to bc.UTXO.
	// The burn owner must therefore be resolved from the evolving
	// candidate scratch state.
	if _, exists :=
		bc.UTXO[string(first.ID)]; exists {
		t.Fatal(
			"intra-block parent unexpectedly exists in canonical UTXO",
		)
	}

	ownerKey, err :=
		bc.resolveCanonicalBurnOwnerKeyAgainstUTXOStateAtHeightV1(
			burn,
			scratch,
			1,
		)
	if err != nil {
		t.Fatalf(
			"resolve intra-block burn owner: %v",
			err,
		)
	}

	want :=
		wallet.HashPubKey(
			middle.PublicKey,
		)

	if !bytes.Equal(
		ownerKey,
		want,
	) {
		t.Fatalf(
			"resolved intra-block owner=%x want=%x",
			ownerKey,
			want,
		)
	}

	if _, err :=
		bc.validateTransactionAgainstUTXOStateV1(
			burn,
			scratch,
			origins,
			createdNormalTx,
			spentInSequence,
			knownTxIDs,
			false,
			1,
		); err != nil {
		t.Fatalf(
			"validate intra-block burn: %v",
			err,
		)
	}
}
