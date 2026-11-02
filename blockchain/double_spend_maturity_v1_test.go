package blockchain

import (
	"strings"
	"testing"
	"time"

	"quantumcoin/wallet"
)

func cloneFeeSpendForDoubleSpendTestV1(
	t *testing.T,
	spend *Transaction,
	sender *wallet.Wallet,
	amount int,
	timestampOffset int64,
) *Transaction {
	t.Helper()

	cloned := &Transaction{
		Inputs: append(
			[]TransactionInput(nil),
			spend.Inputs...,
		),
		Outputs: append(
			[]TransactionOutput(nil),
			spend.Outputs...,
		),
		Timestamp: spend.Timestamp.Add(
			time.Duration(timestampOffset) * time.Second,
		),
		Sender: spend.Sender,
		Amount: float64(amount),
	}

	cloned.Outputs[0].Amount = amount
	cloned.ID = cloned.Hash()

	if err := cloned.Sign(sender.PrivateKey); err != nil {
		t.Fatalf(
			"sign double-spend fixture: %v",
			err,
		)
	}

	return cloned
}

func TestCoinbaseMaturityUsesCandidateSpendHeightV1(
	t *testing.T,
) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	chain.SetCoinbaseMaturity(2)

	if _, err := chain.ValidateTransactionUTXOAtHeightV1(
		spend,
		false,
		1,
	); err == nil ||
		!strings.Contains(
			err.Error(),
			"coinbase output is not mature",
		) {
		t.Fatalf(
			"expected height-1 maturity rejection, got %v",
			err,
		)
	}

	if _, err := chain.ValidateTransactionUTXOAtHeightV1(
		spend,
		false,
		2,
	); err != nil {
		t.Fatalf(
			"maturity boundary spend rejected: %v",
			err,
		)
	}
}

func TestSpendableBalanceUsesNextCandidateHeightV1(
	t *testing.T,
) {
	chain, sender, _, spend := newFeeConsensusTestStateV1(t)
	chain.SetCoinbaseMaturity(2)

	if got := chain.GetSpendableBalance(
		sender.GetAddress(),
	); got != 0 {
		t.Fatalf(
			"immature spendable balance: got %d want 0",
			got,
		)
	}

	bridgeCoinbase := feeTestCoinbaseV1(
		t,
		wallet.NewWallet().GetAddress(),
		50,
	)

	chain.Blocks = append(
		chain.Blocks,
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				bridgeCoinbase,
			},
		},
	)
	chain.UpdateUTXOSet()

	if got := chain.GetSpendableBalance(
		sender.GetAddress(),
	); got != 10 {
		t.Fatalf(
			"maturity-boundary spendable balance: got %d want 10",
			got,
		)
	}

	if err := chain.AddTransaction(spend); err != nil {
		t.Fatalf(
			"maturity-boundary mempool spend rejected: %v",
			err,
		)
	}
}

func TestMempoolReservesOutpointAgainstDoubleSpendV1(
	t *testing.T,
) {
	chain, sender, _, first :=
		newFeeConsensusTestStateV1(t)

	second := cloneFeeSpendForDoubleSpendTestV1(
		t,
		first,
		sender,
		6,
		1,
	)

	if err := chain.AddTransaction(first); err != nil {
		t.Fatalf(
			"add first mempool spend: %v",
			err,
		)
	}

	err := chain.AddTransaction(second)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"input already pending in mempool",
		) {
		t.Fatalf(
			"expected mempool double-spend rejection, got %v",
			err,
		)
	}
}

func TestCanonicalUTXORejectsDuplicateInputInsideTransactionV1(
	t *testing.T,
) {
	chain, sender, _, spend :=
		newFeeConsensusTestStateV1(t)

	duplicateInput := spend.Inputs[0]
	spend.Inputs = append(
		spend.Inputs,
		duplicateInput,
	)

	spend.ID = spend.Hash()

	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf(
			"sign duplicate-input transaction: %v",
			err,
		)
	}

	_, err := chain.ValidateTransactionUTXOV1(
		spend,
		false,
	)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"double spend in validation sequence",
		) {
		t.Fatalf(
			"expected internal double-spend rejection, got %v",
			err,
		)
	}
}

func TestCanonicalUTXORejectsAlreadySpentChainOutpointV1(
	t *testing.T,
) {
	chain, sender, _, first :=
		newFeeConsensusTestStateV1(t)

	miner := wallet.NewWallet()

	chain.Blocks = append(
		chain.Blocks,
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				feeTestCoinbaseV1(
					t,
					miner.GetAddress(),
					53,
				),
				first,
			},
		},
	)
	chain.UpdateUTXOSet()

	second := cloneFeeSpendForDoubleSpendTestV1(
		t,
		first,
		sender,
		6,
		2,
	)

	_, err := chain.ValidateTransactionUTXOV1(
		second,
		false,
	)
	if err == nil ||
		(!strings.Contains(
			err.Error(),
			"missing utxo",
		) &&
			!strings.Contains(
				err.Error(),
				"spent or empty utxo",
			)) {
		t.Fatalf(
			"expected chain double-spend rejection, got %v",
			err,
		)
	}
}

func TestPeerBlockRejectsImmatureCoinbaseSpendV1(
	t *testing.T,
) {
	chain, _, _, spend :=
		newFeeConsensusTestStateV1(t)

	chain.SetCoinbaseMaturity(2)

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
			"mine immature peer block: %v",
			err,
		)
	}

	err = chain.ValidatePeerBlockStrictV2(block)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"coinbase output is not mature",
		) {
		t.Fatalf(
			"expected immature peer spend rejection, got %v",
			err,
		)
	}
}

func TestPeerBlockAcceptsCoinbaseSpendAtMaturityBoundaryV1(
	t *testing.T,
) {
	chain, _, _, spend :=
		newFeeConsensusTestStateV1(t)

	chain.SetCoinbaseMaturity(2)

	miner := wallet.NewWallet()

	blockOneDifficulty := testExpectedNextDifficultyV1(
		t,
		chain,
	)

	blockOne, err := NewBlock(
		1,
		[]*Transaction{
			feeTestCoinbaseV1(
				t,
				miner.GetAddress(),
				50,
			),
		},
		chain.GetLastBlock().Hash,
		miner.GetAddress(),
		blockOneDifficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine maturity bridge block: %v",
			err,
		)
	}

	chain.Blocks = append(
		chain.Blocks,
		blockOne,
	)
	chain.UpdateUTXOSet()

	blockTwoDifficulty := testExpectedNextDifficultyV1(
		t,
		chain,
	)

	blockTwo, err := NewBlock(
		2,
		[]*Transaction{
			feeTestCoinbaseV1(
				t,
				miner.GetAddress(),
				53,
			),
			spend,
		},
		blockOne.Hash,
		miner.GetAddress(),
		blockTwoDifficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine boundary peer block: %v",
			err,
		)
	}

	if err := chain.ValidatePeerBlockStrictV2(
		blockTwo,
	); err != nil {
		t.Fatalf(
			"maturity-boundary peer block rejected: %v",
			err,
		)
	}
}

func TestReplacementChainUsesCandidateHeightForMaturityV1(
	t *testing.T,
) {
	chain, _, _, spend :=
		newFeeConsensusTestStateV1(t)

	chain.SetCoinbaseMaturity(2)

	miner := wallet.NewWallet()
	genesis := chain.GetLastBlock()

	blockOneDifficulty := testExpectedNextDifficultyV1(
		t,
		chain,
	)

	blockOne, err := NewBlock(
		1,
		[]*Transaction{
			feeTestCoinbaseV1(
				t,
				miner.GetAddress(),
				50,
			),
		},
		genesis.Hash,
		miner.GetAddress(),
		blockOneDifficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine replacement bridge block: %v",
			err,
		)
	}

	candidateChain := &Blockchain{
		Blocks: []*Block{
			genesis,
			blockOne,
		},
		TotalSupply:      chain.TotalSupply,
		coinbaseMaturity: chain.coinbaseMaturity,
		activationPolicy: chain.activationPolicy,
	}

	blockTwoDifficulty := testExpectedNextDifficultyV1(
		t,
		candidateChain,
	)

	blockTwo, err := NewBlock(
		2,
		[]*Transaction{
			feeTestCoinbaseV1(
				t,
				miner.GetAddress(),
				53,
			),
			spend,
		},
		blockOne.Hash,
		miner.GetAddress(),
		blockTwoDifficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine replacement boundary block: %v",
			err,
		)
	}

	incoming := []*Block{
		genesis,
		blockOne,
		blockTwo,
	}

	if err := chain.ValidateReplacementChainStrictV1(
		incoming,
	); err != nil {
		t.Fatalf(
			"maturity-boundary replacement chain rejected: %v",
			err,
		)
	}
}
