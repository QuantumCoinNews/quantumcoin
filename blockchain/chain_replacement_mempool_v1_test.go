package blockchain

import (
	"testing"

	"quantumcoin/wallet"
)

func TestReplaceChainRemovesConfirmedPendingTransactionV1(
	t *testing.T,
) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)

	if err := chain.AddTransaction(spend); err != nil {
		t.Fatalf(
			"add pending transaction: %v",
			err,
		)
	}

	if got := len(chain.PendingTxs()); got != 1 {
		t.Fatalf(
			"pending transaction count before replacement: got %d want 1",
			got,
		)
	}

	miner := wallet.NewWallet()
	genesis := chain.GetLastBlock()

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
		genesis.Hash,
		miner.GetAddress(),
		difficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine replacement block: %v",
			err,
		)
	}

	incoming := []*Block{
		genesis,
		block,
	}

	if err := chain.ReplaceChain(incoming); err != nil {
		t.Fatalf(
			"replace chain: %v",
			err,
		)
	}

	if got := len(chain.PendingTxs()); got != 0 {
		t.Fatalf(
			"confirmed transaction remained pending after replacement: got %d",
			got,
		)
	}
}
