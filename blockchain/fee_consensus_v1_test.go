package blockchain

import (
	"strings"
	"testing"
	"time"

	"quantumcoin/wallet"
)

func newFeeConsensusTestStateV1(
	t *testing.T,
) (*Blockchain, *wallet.Wallet, *wallet.Wallet, *Transaction) {
	t.Helper()

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

	genesis := &Block{
		Index:        0,
		Timestamp:    CanonicalGenesisUnixV1,
		Transactions: []*Transaction{funding},
		PrevHash:     []byte{},
		Hash:         []byte("fee-test-genesis-hash"),
		Nonce:        0,
		Miner:        "genesis",
		Difficulty:   1,
		Metadata:     map[string]string{},
	}

	chain := &Blockchain{
		Blocks:           []*Block{genesis},
		UTXO:             map[string][]TransactionOutput{},
		TotalSupply:      1_000,
		coinbaseMaturity: 0,
		pendingTxs:       []*Transaction{},
	}
	chain.UpdateUTXOSet()

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
			CanonicalGenesisUnixV1+30,
			0,
		).UTC(),
		Sender: sender.GetAddress(),
		Amount: 7,
	}
	spend.ID = spend.Hash()

	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf(
			"sign fee test transaction: %v",
			err,
		)
	}

	return chain, sender, receiver, spend
}

func feeTestCoinbaseV1(
	t *testing.T,
	miner string,
	amount int,
) *Transaction {
	t.Helper()

	coinbase, err := newCoinbaseTxWithRewardV1(
		miner,
		amount,
		time.Unix(
			CanonicalGenesisUnixV1+30,
			0,
		).UTC(),
	)
	if err != nil {
		t.Fatalf(
			"create fee test coinbase: %v",
			err,
		)
	}

	return coinbase
}

func TestCalculateBlockFeesV1UsesVerifiedUTXOValue(
	t *testing.T,
) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)

	fees, err := chain.CalculateBlockFeesV1(
		[]*Transaction{
			spend,
		},
	)
	if err != nil {
		t.Fatalf(
			"calculate verified fees: %v",
			err,
		)
	}

	if fees != 3 {
		t.Fatalf(
			"verified fees: got %d want 3",
			fees,
		)
	}
}

func TestValidateBlockRewardWithFeesV1(
	t *testing.T,
) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	miner := wallet.NewWallet()

	block := &Block{
		Index: 1,
		Transactions: []*Transaction{
			feeTestCoinbaseV1(
				t,
				miner.GetAddress(),
				53,
			),
			spend,
		},
	}

	result, err := chain.ValidateBlockRewardWithFeesV1(block)
	if err != nil {
		t.Fatalf(
			"valid subsidy plus fees rejected: %v",
			err,
		)
	}

	if result.Subsidy != 50 ||
		result.Fees != 3 ||
		result.CoinbaseTotal != 53 {
		t.Fatalf(
			"unexpected reward breakdown: %+v",
			result,
		)
	}

	block.Transactions[0].Outputs[0].Amount = 54

	if _, err := chain.ValidateBlockRewardWithFeesV1(
		block,
	); err == nil {
		t.Fatal(
			"expected excessive coinbase rejection",
		)
	}
}

func TestTotalMintedExcludesTransactionFeesV1(
	t *testing.T,
) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	miner := wallet.NewWallet()

	block := &Block{
		Index: 1,
		Transactions: []*Transaction{
			feeTestCoinbaseV1(
				t,
				miner.GetAddress(),
				53,
			),
			spend,
		},
	}

	chain.Blocks = append(
		chain.Blocks,
		block,
	)

	if got := chain.TotalMinted(); got != 60 {
		t.Fatalf(
			"minted subsidy total: got %d want 60",
			got,
		)
	}
}

func TestCalculateBlockFeesV1RejectsMissingPrevout(
	t *testing.T,
) {
	chain, sender, _, spend := newFeeConsensusTestStateV1(t)

	spend.Inputs[0].TxID = []byte(
		"missing-prevout",
	)

	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf(
			"re-sign missing prevout transaction: %v",
			err,
		)
	}

	_, err := chain.CalculateBlockFeesV1(
		[]*Transaction{
			spend,
		},
	)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"missing utxo",
		) {
		t.Fatalf(
			"expected missing UTXO rejection, got %v",
			err,
		)
	}
}

func TestMineBlockPaysVerifiedFeesAndDoesNotMintThemV1(
	t *testing.T,
) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	miner := wallet.NewWallet()

	if err := chain.AddTransaction(spend); err != nil {
		t.Fatalf(
			"add fee transaction: %v",
			err,
		)
	}

	block, err := chain.MineBlock(
		miner.GetAddress(),
		1,
	)
	if err != nil {
		t.Fatalf(
			"mine fee-bearing block: %v",
			err,
		)
	}

	_, coinbaseTotal, err := blockCoinbaseTotalV1(
		block,
		false,
	)
	if err != nil {
		t.Fatalf(
			"read mined coinbase: %v",
			err,
		)
	}

	if coinbaseTotal != 53 {
		t.Fatalf(
			"mined coinbase: got %d want 53",
			coinbaseTotal,
		)
	}

	if got := chain.TotalMinted(); got != 60 {
		t.Fatalf(
			"minted after fee block: got %d want 60",
			got,
		)
	}
}

func TestPeerBlockValidationUsesVerifiedFeesV1(
	t *testing.T,
) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	miner := wallet.NewWallet()
	previous := chain.GetLastBlock()

	difficulty := testExpectedNextDifficultyV1(
		t,
		chain,
	)

	block, err := NewBlock(
		1,
		[]*Transaction{
			feeTestCoinbaseV1(
				t,
				miner.GetAddress(),
				53,
			),
			spend,
		},
		previous.Hash,
		miner.GetAddress(),
		difficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine peer fee block: %v",
			err,
		)
	}

	if err := chain.ValidatePeerBlockStrictV2(
		block,
	); err != nil {
		t.Fatalf(
			"peer fee block rejected: %v",
			err,
		)
	}
}

func TestReplacementChainValidationUsesVerifiedFeesV1(
	t *testing.T,
) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	miner := wallet.NewWallet()
	previous := chain.GetLastBlock()

	difficulty := testExpectedNextDifficultyV1(
		t,
		chain,
	)

	block, err := NewBlock(
		1,
		[]*Transaction{
			feeTestCoinbaseV1(
				t,
				miner.GetAddress(),
				53,
			),
			spend,
		},
		previous.Hash,
		miner.GetAddress(),
		difficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine replacement fee block: %v",
			err,
		)
	}

	incoming := []*Block{
		previous,
		block,
	}

	if err := chain.ValidateReplacementChainStrictV1(
		incoming,
	); err != nil {
		t.Fatalf(
			"replacement fee chain rejected: %v",
			err,
		)
	}
}
