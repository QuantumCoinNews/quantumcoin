package blockchain

import (
	"bytes"
	"testing"

	"quantumcoin/wallet"
)

func TestResolveValidatedHistoricalBurnOwnerKeyV1(
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

	if err := tx.Sign(
		owner.PrivateKey,
	); err != nil {
		t.Fatalf(
			"sign canonical burn transaction: %v",
			err,
		)
	}

	if _, err :=
		bc.ValidateTransactionUTXOAtHeightV1(
			tx,
			false,
			1,
		); err != nil {
		t.Fatalf(
			"canonical validation before historical attribution: %v",
			err,
		)
	}

	got, err :=
		ResolveValidatedHistoricalBurnOwnerKeyV1(
			tx,
		)
	if err != nil {
		t.Fatalf(
			"resolve validated historical burn owner: %v",
			err,
		)
	}

	want :=
		wallet.HashPubKey(
			owner.PublicKey,
		)

	if !bytes.Equal(
		got,
		want,
	) {
		t.Fatalf(
			"historical owner=%x want=%x",
			got,
			want,
		)
	}
}

func TestResolveValidatedHistoricalBurnOwnerRejectsMixedInputOwnersV1(
	t *testing.T,
) {
	first := wallet.NewWallet()
	second := wallet.NewWallet()

	if first == nil || second == nil {
		t.Fatal(
			"create wallets returned nil",
		)
	}

	tx := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     []byte{0x01},
				OutIndex: 0,
				PubKey: append(
					[]byte(nil),
					first.PublicKey...,
				),
			},
			{
				TxID:     []byte{0x02},
				OutIndex: 0,
				PubKey: append(
					[]byte(nil),
					second.PublicKey...,
				),
			},
		},
		Outputs: []TransactionOutput{
			NewCanonicalBurnOutputV1(
				1,
			),
		},
	}

	if _, err :=
		ResolveValidatedHistoricalBurnOwnerKeyV1(
			tx,
		); err == nil {
		t.Fatal(
			"mixed-owner historical burn unexpectedly accepted",
		)
	}
}
