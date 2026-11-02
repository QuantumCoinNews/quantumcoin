package blockchain

import (
	"encoding/hex"
	"testing"
	"time"

	qconsensus "quantumcoin/consensus"
	"quantumcoin/wallet"
)

func atomRewardTestChainV1(
	t *testing.T,
	activationHeight int,
) *Blockchain {
	t.Helper()

	bc := NewBlockchain(
		CanonicalInitialMiningRewardV1,
		int(QCTotalSupplyHardCap),
	)
	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	bc.SetCoinbaseMaturity(0)

	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil {
		t.Fatal("test chain has no genesis")
	}

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "atom-reward-test",
			LegacyGenesisHex: hex.EncodeToString(bc.Blocks[0].Hash),
			ActivationHeight: activationHeight,
			PreserveHistory:  true,
			PreserveBalances: true,
			TargetProtocol:   qconsensus.ProtocolAtomV3,
		},
	}

	if err := bc.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf("bind AtomV3 activation: %v", err)
	}

	return bc
}

func atomRewardOneAtomFeeFixtureV1(
	t *testing.T,
	bc *Blockchain,
) (*Transaction, *wallet.Wallet) {
	t.Helper()

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
			CanonicalGenesisUnixV1+30,
			0,
		).UTC(),
		Sender: sender.GetAddress(),
		Amount: 999_999,
	}

	spend.ID = spend.Hash()

	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("sign AtomV3 spend: %v", err)
	}

	return spend, receiver
}

func TestAtomV3BlockRewardAcceptsOneAtomVerifiedFeeV1(
	t *testing.T,
) {
	bc := atomRewardTestChainV1(t, 1)

	spend, _ := atomRewardOneAtomFeeFixtureV1(t, bc)
	miner := wallet.NewWallet()

	coinbase, err := newCoinbaseTxWithRewardV1(
		miner.GetAddress(),
		50_000_001,
		time.Unix(
			CanonicalGenesisUnixV1+30,
			0,
		).UTC(),
	)
	if err != nil {
		t.Fatalf("create AtomV3 coinbase: %v", err)
	}

	block := &Block{
		Index: 1,
		Transactions: []*Transaction{
			coinbase,
			spend,
		},
	}

	result, err := bc.ValidateBlockRewardAgainstStateV1(
		block,
		0,
	)
	if err != nil {
		t.Fatalf("validate AtomV3 block reward: %v", err)
	}

	if result.Subsidy != 50_000_000 {
		t.Fatalf(
			"subsidy=%d want=50000000",
			result.Subsidy,
		)
	}

	if result.Fees != 1 {
		t.Fatalf(
			"fees=%d want=1",
			result.Fees,
		)
	}

	if result.CoinbaseTotal != 50_000_001 {
		t.Fatalf(
			"coinbase=%d want=50000001",
			result.CoinbaseTotal,
		)
	}
}

func TestAtomV3BlockRewardRejectsMixedWholeQCAndAtomUnitsV1(
	t *testing.T,
) {
	bc := atomRewardTestChainV1(t, 1)

	spend, _ := atomRewardOneAtomFeeFixtureV1(t, bc)
	miner := wallet.NewWallet()

	// Eski hatalı mantık:
	// 50 whole-QC integer + 1 atom = 51
	coinbase, err := newCoinbaseTxWithRewardV1(
		miner.GetAddress(),
		51,
		time.Unix(
			CanonicalGenesisUnixV1+30,
			0,
		).UTC(),
	)
	if err != nil {
		t.Fatalf("create mixed-unit coinbase: %v", err)
	}

	block := &Block{
		Index: 1,
		Transactions: []*Transaction{
			coinbase,
			spend,
		},
	}

	if _, err := bc.ValidateBlockRewardAgainstStateV1(
		block,
		0,
	); err == nil {
		t.Fatal(
			"mixed whole-QC subsidy plus atom fee unexpectedly accepted",
		)
	}
}

func TestLegacyBlockRewardPathRemainsWholeQCV1(
	t *testing.T,
) {
	bc := atomRewardTestChainV1(t, 2)

	miner := wallet.NewWallet()

	coinbase, err := newCoinbaseTxWithRewardV1(
		miner.GetAddress(),
		50,
		time.Unix(
			CanonicalGenesisUnixV1+30,
			0,
		).UTC(),
	)
	if err != nil {
		t.Fatalf("create legacy coinbase: %v", err)
	}

	block := &Block{
		Index:        1,
		Transactions: []*Transaction{coinbase},
	}

	result, err := bc.ValidateBlockRewardAgainstStateV1(
		block,
		0,
	)
	if err != nil {
		t.Fatalf("validate legacy block reward: %v", err)
	}

	if result.Subsidy != 50 ||
		result.Fees != 0 ||
		result.CoinbaseTotal != 50 {
		t.Fatalf(
			"legacy reward changed: %+v",
			result,
		)
	}
}
