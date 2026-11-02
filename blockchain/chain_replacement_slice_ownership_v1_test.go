package blockchain

import "testing"

func TestReplaceChainCopiesIncomingBlockSliceV1(t *testing.T) {
	chain := NewBlockchain(1, 0)
	previous := chain.GetLastBlock()

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	difficulty := testExpectedNextDifficultyV1(
		t,
		chain,
	)

	next, err := NewBlock(
		previous.Index+1,
		[]*Transaction{
			testReplacementCoinbaseTxV1(reward),
		},
		previous.Hash,
		"replacement-miner",
		difficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine replacement block: %v",
			err,
		)
	}

	incoming := append(
		[]*Block(nil),
		chain.GetAllBlocks()...,
	)
	incoming = append(
		incoming,
		next,
	)

	if err := chain.ReplaceChain(incoming); err != nil {
		t.Fatalf(
			"replace valid chain: %v",
			err,
		)
	}

	incoming[1] = nil

	if len(chain.Blocks) != 2 {
		t.Fatalf(
			"replacement chain length = %d, want 2",
			len(chain.Blocks),
		)
	}

	if chain.Blocks[1] == nil {
		t.Fatal(
			"external incoming slice mutation changed accepted chain",
		)
	}
}
