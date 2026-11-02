package blockchain

import (
	"strings"
	"testing"
)

func testLongerReplacementChainWithDifferentGenesisV1(
	t *testing.T,
) []*Block {
	t.Helper()

	alternative := NewBlockchain(2, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	previous := alternative.GetLastBlock()

	next, err := NewBlock(
		previous.Index+1,
		[]*Transaction{
			testReplacementCoinbaseTxV1(reward),
		},
		previous.Hash,
		"replacement-miner",
		1,
	)
	if err != nil {
		t.Fatalf("mine alternative replacement block: %v", err)
	}

	blocks := append(
		[]*Block(nil),
		alternative.GetAllBlocks()...,
	)

	return append(blocks, next)
}

func TestReplaceChainDirectRejectsGenesisMismatchV1(
	t *testing.T,
) {
	chain := NewBlockchain(1, 0)
	incoming := testLongerReplacementChainWithDifferentGenesisV1(t)

	err := chain.ReplaceChain(incoming)
	if err == nil {
		t.Fatal("direct ReplaceChain accepted mismatched genesis")
	}

	if !strings.Contains(err.Error(), "genesis") {
		t.Fatalf(
			"expected genesis mismatch error, got: %v",
			err,
		)
	}

	if got := chain.GetBestHeight(); got != 0 {
		t.Fatalf(
			"rejected replacement changed height: got %d, want 0",
			got,
		)
	}
}

func TestReplaceChainDirectRejectsNilBlockchainV1(
	t *testing.T,
) {
	source := NewBlockchain(1, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	previous := source.GetLastBlock()

	next, err := NewBlock(
		previous.Index+1,
		[]*Transaction{
			testReplacementCoinbaseTxV1(reward),
		},
		previous.Hash,
		"replacement-miner",
		1,
	)
	if err != nil {
		t.Fatalf("mine replacement block: %v", err)
	}

	incoming := append(
		[]*Block(nil),
		source.GetAllBlocks()...,
	)
	incoming = append(incoming, next)

	var chain *Blockchain

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf(
				"direct ReplaceChain panicked for nil blockchain: %v",
				recovered,
			)
		}
	}()

	err = chain.ReplaceChain(incoming)
	if err == nil {
		t.Fatal("nil blockchain accepted replacement chain")
	}
}
