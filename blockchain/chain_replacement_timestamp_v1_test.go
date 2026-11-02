package blockchain

import (
	"strings"
	"testing"
)

func TestValidateReplacementChainStrictV1RejectsBackwardTimestamp(
	t *testing.T,
) {
	chain := NewBlockchain(1, 0)
	previous := chain.GetLastBlock()

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

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

	next.Timestamp = previous.Timestamp - 1

	pow := NewProofOfWork(next)
	if pow == nil {
		t.Fatal("replacement proof of work is unavailable")
	}

	nonce, hash, err := pow.Run()
	if err != nil {
		t.Fatalf("remine replacement block: %v", err)
	}

	next.Nonce = nonce
	next.Hash = append([]byte(nil), hash...)

	incoming := append(
		[]*Block(nil),
		chain.GetAllBlocks()...,
	)
	incoming = append(incoming, next)

	err = chain.ValidateReplacementChainStrictV1(incoming)
	if err == nil {
		t.Fatal(
			"replacement validation accepted backward block timestamp",
		)
	}

	message := strings.ToLower(err.Error())
	if !strings.Contains(message, "time") &&
		!strings.Contains(message, "timestamp") {
		t.Fatalf(
			"expected timestamp ordering error, got: %v",
			err,
		)
	}
}
