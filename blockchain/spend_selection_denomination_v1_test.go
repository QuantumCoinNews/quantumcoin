package blockchain

import (
	"testing"

	"quantumcoin/wallet"
)

func atomSpendSelectionTestChainV1(
	t *testing.T,
	activationHeight int,
) (*Blockchain, *wallet.Wallet) {
	t.Helper()

	bc := atomDenominationTestChainV1(
		t,
		activationHeight,
	)

	owner := wallet.NewWallet()

	funding := &Transaction{
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     1,
				PubKeyHash: wallet.HashPubKey(owner.PublicKey),
			},
		},
		Sender: "COINBASE",
		Amount: 1,
	}

	funding.ID = funding.Hash()

	bc.Blocks[0].Transactions = []*Transaction{
		funding,
	}

	bc.coinbaseMaturity = 0
	bc.UpdateUTXOSet()

	return bc, owner
}

func TestSpendSelectionLegacyAmountRemainsWholeQCV1(
	t *testing.T,
) {
	bc, owner := atomSpendSelectionTestChainV1(
		t,
		100,
	)

	_, accumulated, err :=
		bc.FindSpendableOutputsAtHeightV1(
			wallet.HashPubKey(owner.PublicKey),
			1,
			99,
		)

	if err != nil {
		t.Fatalf("legacy selection: %v", err)
	}

	if accumulated != 1 {
		t.Fatalf(
			"legacy accumulated=%d want=1",
			accumulated,
		)
	}
}

func TestSpendSelectionNormalizesLegacyUTXOAtAtomActivationV1(
	t *testing.T,
) {
	bc, owner := atomSpendSelectionTestChainV1(
		t,
		1,
	)

	_, accumulated, err :=
		bc.FindSpendableOutputsAtHeightV1(
			wallet.HashPubKey(owner.PublicKey),
			1,
			1,
		)

	if err != nil {
		t.Fatalf("AtomV3 selection: %v", err)
	}

	if accumulated != 1_000_000 {
		t.Fatalf(
			"AtomV3 accumulated=%d want=1000000",
			accumulated,
		)
	}
}

func TestSpendSelectionRejectsInvalidHeightV1(
	t *testing.T,
) {
	bc, owner := atomSpendSelectionTestChainV1(
		t,
		1,
	)

	if _, _, err :=
		bc.FindSpendableOutputsAtHeightV1(
			wallet.HashPubKey(owner.PublicKey),
			1,
			-1,
		); err == nil {
		t.Fatal(
			"negative spend height unexpectedly accepted",
		)
	}
}
