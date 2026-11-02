package blockchain

import (
	"encoding/hex"
	"testing"
	"time"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

func bindAtomActivationForUTXOTestV1(
	t *testing.T,
	bc *Blockchain,
	height int,
) {
	t.Helper()

	if bc == nil || len(bc.Blocks) == 0 || bc.Blocks[0] == nil {
		t.Fatal("test blockchain has no genesis block")
	}

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "atom-utxo-test",
			LegacyGenesisHex: hex.EncodeToString(bc.Blocks[0].Hash),
			ActivationHeight: height,
			PreserveHistory:  true,
			PreserveBalances: true,
			TargetProtocol:   qconsensus.ProtocolAtomV3,
		},
	}

	if err := bc.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf("bind AtomV3 policy: %v", err)
	}
}

func TestAtomV3UTXOValidationRepresentsOneAtomFeeV1(
	t *testing.T,
) {
	bc := NewBlockchain(
		CanonicalInitialMiningRewardV1,
		int(QCTotalSupplyHardCap),
	)
	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	bc.SetCoinbaseMaturity(0)

	sender := wallet.NewWallet()
	receiver := wallet.NewWallet()

	funding := &Transaction{
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     1,
				PubKeyHash: wallet.HashPubKey(sender.PublicKey),
			},
		},
		Timestamp: CanonicalGenesisTimeV1(),
		Sender:    "COINBASE",
		Amount:    1,
	}
	funding.ID = funding.Hash()

	bc.Blocks[0].Transactions = []*Transaction{funding}
	bc.UpdateUTXOSet()

	bindAtomActivationForUTXOTestV1(
		t,
		bc,
		1,
	)

	spend := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     append([]byte(nil), funding.ID...),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount:     999_999,
				PubKeyHash: wallet.HashPubKey(receiver.PublicKey),
			},
		},
		Timestamp: time.Unix(
			CanonicalGenesisUnixV1+60,
			0,
		).UTC(),
		Sender: sender.GetAddress(),
		Amount: 999_999,
	}

	spend.ID = spend.Hash()

	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("sign AtomV3 spend: %v", err)
	}

	result, err := bc.ValidateTransactionUTXOAtHeightV1(
		spend,
		false,
		1,
	)
	if err != nil {
		t.Fatalf("validate AtomV3 spend: %v", err)
	}

	if result.InputTotal != 1_000_000 {
		t.Fatalf(
			"input total=%d want=1000000 atoms",
			result.InputTotal,
		)
	}

	if result.OutputTotal != 999_999 {
		t.Fatalf(
			"output total=%d want=999999 atoms",
			result.OutputTotal,
		)
	}

	if result.Fee != 1 {
		t.Fatalf(
			"fee=%d want=1 atom",
			result.Fee,
		)
	}

	if got := qconsensus.FormatAtomsAsQCV1(
		int64(result.Fee),
	); got != "0.000001" {
		t.Fatalf(
			"fee display=%q want=%q",
			got,
			"0.000001",
		)
	}
}

func TestLegacyUTXOValidationStillUsesWholeQCBeforeActivationV1(
	t *testing.T,
) {
	bc := NewBlockchain(
		CanonicalInitialMiningRewardV1,
		int(QCTotalSupplyHardCap),
	)
	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	bc.SetCoinbaseMaturity(0)

	sender := wallet.NewWallet()
	receiver := wallet.NewWallet()

	funding := &Transaction{
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     10,
				PubKeyHash: wallet.HashPubKey(sender.PublicKey),
			},
		},
		Timestamp: CanonicalGenesisTimeV1(),
		Sender:    "COINBASE",
		Amount:    10,
	}
	funding.ID = funding.Hash()

	bc.Blocks[0].Transactions = []*Transaction{funding}
	bc.UpdateUTXOSet()

	bindAtomActivationForUTXOTestV1(
		t,
		bc,
		2,
	)

	spend := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     append([]byte(nil), funding.ID...),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount:     7,
				PubKeyHash: wallet.HashPubKey(receiver.PublicKey),
			},
		},
		Timestamp: time.Unix(
			CanonicalGenesisUnixV1+60,
			0,
		).UTC(),
		Sender: sender.GetAddress(),
		Amount: 7,
	}

	spend.ID = spend.Hash()

	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("sign legacy spend: %v", err)
	}

	result, err := bc.ValidateTransactionUTXOAtHeightV1(
		spend,
		false,
		1,
	)
	if err != nil {
		t.Fatalf("validate legacy spend: %v", err)
	}

	if result.InputTotal != 10 ||
		result.OutputTotal != 7 ||
		result.Fee != 3 {
		t.Fatalf(
			"legacy totals changed: %+v",
			result,
		)
	}
}

func TestAtomV3IntraBlockOutputIsNotDoubleConvertedV1(
	t *testing.T,
) {
	bc := NewBlockchain(
		CanonicalInitialMiningRewardV1,
		int(QCTotalSupplyHardCap),
	)
	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	bc.SetCoinbaseMaturity(0)

	sender := wallet.NewWallet()
	middle := wallet.NewWallet()
	finalReceiver := wallet.NewWallet()

	funding := &Transaction{
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     1,
				PubKeyHash: wallet.HashPubKey(sender.PublicKey),
			},
		},
		Timestamp: CanonicalGenesisTimeV1(),
		Sender:    "COINBASE",
		Amount:    1,
	}
	funding.ID = funding.Hash()

	bc.Blocks[0].Transactions = []*Transaction{funding}
	bc.UpdateUTXOSet()

	bindAtomActivationForUTXOTestV1(
		t,
		bc,
		1,
	)

	first := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     append([]byte(nil), funding.ID...),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount:     999_999,
				PubKeyHash: wallet.HashPubKey(middle.PublicKey),
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

	if err := first.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("sign first AtomV3 spend: %v", err)
	}

	second := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     append([]byte(nil), first.ID...),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount:     999_998,
				PubKeyHash: wallet.HashPubKey(finalReceiver.PublicKey),
			},
		},
		Timestamp: time.Unix(
			CanonicalGenesisUnixV1+61,
			0,
		).UTC(),
		Sender: middle.GetAddress(),
		Amount: 999_998,
	}
	second.ID = second.Hash()

	if err := second.Sign(middle.PrivateKey); err != nil {
		t.Fatalf("sign second AtomV3 spend: %v", err)
	}

	fees, err := bc.ValidateTransactionsUTXOAtHeightV1(
		[]*Transaction{
			first,
			second,
		},
		1,
	)
	if err != nil {
		t.Fatalf(
			"validate AtomV3 intra-block spends: %v",
			err,
		)
	}

	if fees != 2 {
		t.Fatalf(
			"AtomV3 intra-block total fees=%d want=2 atoms",
			fees,
		)
	}
}
