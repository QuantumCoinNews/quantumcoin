package blockchain

import (
	"testing"
	"time"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

func TestValidateTransactionsUTXORejectsCanonicalBurnAbovePerTransactionLimitV1(
	t *testing.T,
) {
	bc, owner := canonicalBurnLimitTestChainV1(t)

	tx :=
		canonicalBurnForQCAmountV1(
			t,
			bc,
			owner,
			"1000.000001",
		)

	if _, err :=
		bc.ValidateTransactionsUTXOAtHeightV1(
			[]*Transaction{tx},
			1,
		); err == nil {
		t.Fatal(
			"UTXO candidate sequence accepted canonical burn above per-transaction maximum",
		)
	}
}

func TestValidateTransactionsUTXORejectsCandidateOwnerLifetimeBurnAboveLimitV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 1)

	owner := wallet.NewWallet()
	if owner == nil {
		t.Fatal("create candidate burn owner returned nil")
	}

	funding := &Transaction{
		Inputs: []TransactionInput{},
		Sender: "COINBASE",
	}

	for i := 0; i < 11; i++ {
		funding.Outputs = append(
			funding.Outputs,
			TransactionOutput{
				Amount:     1000,
				PubKeyHash: wallet.HashPubKey(owner.PublicKey),
			},
		)
	}

	funding.ID = funding.Hash()

	bc.Blocks[0].Transactions = []*Transaction{
		funding,
	}
	bc.coinbaseMaturity = 0
	bc.UpdateUTXOSet()

	if got := bc.ProtocolAtHeightV1(1); got != qconsensus.ProtocolAtomV3 {
		t.Fatalf(
			"candidate burn fixture protocol=%v want=%v",
			got,
			qconsensus.ProtocolAtomV3,
		)
	}

	transactions := make(
		[]*Transaction,
		0,
		11,
	)

	for outIndex := 0; outIndex < 11; outIndex++ {
		tx := &Transaction{
			Inputs: []TransactionInput{
				{
					TxID: append(
						[]byte(nil),
						funding.ID...,
					),
					OutIndex: outIndex,
				},
			},
			Outputs: []TransactionOutput{
				NewCanonicalBurnOutputV1(
					1_000_000_000,
				),
			},
			Sender: owner.GetAddress(),
			Timestamp: time.Unix(
				CanonicalGenesisUnixV1+int64(outIndex+1),
				0,
			).UTC(),
		}

		tx.ID = tx.Hash()

		if err := tx.Sign(
			owner.PrivateKey,
		); err != nil {
			t.Fatalf(
				"sign candidate burn[%d]: %v",
				outIndex,
				err,
			)
		}

		if !tx.Verify() {
			t.Fatalf(
				"candidate burn[%d] failed verification",
				outIndex,
			)
		}

		transactions = append(
			transactions,
			tx,
		)
	}

	if _, err :=
		bc.ValidateTransactionsUTXOAtHeightV1(
			transactions,
			1,
		); err == nil {
		t.Fatal(
			"candidate sequence accepted owner burn above lifetime maximum",
		)
	}
}

func TestValidateTransactionsUTXORejectsCandidateNetworkLifetimeBurnAboveLimitV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 1)

	candidateOwner := wallet.NewWallet()
	if candidateOwner == nil {
		t.Fatal("create candidate network burn owner returned nil")
	}

	historicalOwner := wallet.NewWallet()
	if historicalOwner == nil {
		t.Fatal("create historical network burn owner returned nil")
	}

	funding := &Transaction{
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     1,
				PubKeyHash: wallet.HashPubKey(candidateOwner.PublicKey),
			},
			{
				Amount:     1,
				PubKeyHash: wallet.HashPubKey(candidateOwner.PublicKey),
			},
		},
		Sender: "COINBASE",
		Amount: 2,
	}
	funding.ID = funding.Hash()

	bc.Blocks[0].Transactions = []*Transaction{
		funding,
	}
	bc.coinbaseMaturity = 0

	historicalBurn :=
		historicalBurnForOwnerV1(
			t,
			historicalOwner,
			1_274_999_000_000,
			0xD1,
		)

	bc.Blocks = append(
		bc.Blocks,
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				historicalBurn,
			},
		},
	)

	bc.UpdateUTXOSet()

	if got := bc.ProtocolAtHeightV1(2); got != qconsensus.ProtocolAtomV3 {
		t.Fatalf(
			"candidate network burn protocol=%v want=%v",
			got,
			qconsensus.ProtocolAtomV3,
		)
	}

	historicalTotal, err := bc.TotalBurnedAtomsCheckedV1()
	if err != nil {
		t.Fatalf(
			"historical network burn accounting: %v",
			err,
		)
	}

	if historicalTotal != 1_274_999_000_000 {
		t.Fatalf(
			"historical network burned=%d want=%d atoms",
			historicalTotal,
			int64(1_274_999_000_000),
		)
	}

	transactions := make(
		[]*Transaction,
		0,
		2,
	)

	for outIndex := 0; outIndex < 2; outIndex++ {
		tx := &Transaction{
			Inputs: []TransactionInput{
				{
					TxID: append(
						[]byte(nil),
						funding.ID...,
					),
					OutIndex: outIndex,
				},
			},
			Outputs: []TransactionOutput{
				NewCanonicalBurnOutputV1(
					1_000_000,
				),
			},
			Sender: candidateOwner.GetAddress(),
			Timestamp: time.Unix(
				CanonicalGenesisUnixV1+int64(outIndex+101),
				0,
			).UTC(),
		}

		tx.ID = tx.Hash()

		if err := tx.Sign(candidateOwner.PrivateKey); err != nil {
			t.Fatalf(
				"sign candidate network burn[%d]: %v",
				outIndex,
				err,
			)
		}

		if !tx.Verify() {
			t.Fatalf(
				"candidate network burn[%d] signature verification failed",
				outIndex,
			)
		}

		transactions = append(
			transactions,
			tx,
		)
	}

	if _, err :=
		bc.ValidateTransactionsUTXOAtHeightV1(
			transactions,
			2,
		); err == nil {
		t.Fatal(
			"candidate sequence accepted network burn above lifetime maximum",
		)
	}
}
