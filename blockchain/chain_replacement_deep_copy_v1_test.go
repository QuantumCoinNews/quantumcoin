package blockchain

import (
	"bytes"
	"testing"

	"quantumcoin/wallet"
)

func TestReplaceChainDeepCopiesAcceptedBlocksV1(t *testing.T) {
	chain, _, _, _ := newFeeConsensusTestStateV1(t)

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
				50,
			),
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

	acceptedIndex := chain.Blocks[1].Index
	acceptedHash := append(
		[]byte(nil),
		chain.Blocks[1].Hash...,
	)
	acceptedMiner := chain.Blocks[1].Miner
	acceptedReward :=
		chain.Blocks[1].
			Transactions[0].
			Outputs[0].
			Amount

	incoming[1].Index = acceptedIndex + 100
	incoming[1].Hash[0] ^= 0xff
	incoming[1].Miner = "external-mutation"
	incoming[1].
		Transactions[0].
		Outputs[0].
		Amount++

	if chain.Blocks[1].Index != acceptedIndex {
		t.Fatalf(
			"external block pointer mutation changed accepted index",
		)
	}

	if !bytes.Equal(
		chain.Blocks[1].Hash,
		acceptedHash,
	) {
		t.Fatalf(
			"external block hash mutation changed accepted chain",
		)
	}

	if chain.Blocks[1].Miner != acceptedMiner {
		t.Fatalf(
			"external block miner mutation changed accepted chain",
		)
	}

	if chain.Blocks[1].
		Transactions[0].
		Outputs[0].
		Amount != acceptedReward {
		t.Fatalf(
			"external transaction mutation changed accepted chain",
		)
	}
}
