package blockchain

import (
	"strings"
	"testing"
	"time"
)

func testReplacementCoinbaseTxV1(amount int) *Transaction {
	tx := &Transaction{
		ID:     nil,
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     amount,
				PubKeyHash: []byte("replacement-miner"),
			},
		},
		Timestamp: time.Now(),
		Sender:    "COINBASE",
		Amount:    float64(amount),
	}

	tx.ID = tx.Hash()

	return tx
}

func TestValidateReplacementChainStrictV1RejectsNotLonger(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	err := bc.ValidateReplacementChainStrictV1(
		bc.GetAllBlocks(),
	)
	if err == nil {
		t.Fatal(
			"expected not-longer replacement rejection",
		)
	}
}

func TestValidateReplacementChainStrictV1RejectsGenesisMismatch(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)
	alt := NewBlockchain(2, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := alt.GetLastBlock()

	difficulty := testExpectedNextDifficultyV1(
		t,
		alt,
	)

	next, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testReplacementCoinbaseTxV1(reward),
		},
		prev.Hash,
		"replacement-miner",
		difficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine replacement test block: %v",
			err,
		)
	}

	blocks := append(
		[]*Block(nil),
		alt.GetAllBlocks()...,
	)
	blocks = append(
		blocks,
		next,
	)

	err = bc.ValidateReplacementChainStrictV1(
		blocks,
	)
	if err == nil {
		t.Fatal(
			"expected genesis mismatch rejection",
		)
	}

	if !strings.Contains(
		err.Error(),
		"genesis",
	) {
		t.Fatalf(
			"expected genesis mismatch error, got: %v",
			err,
		)
	}
}

func TestReplaceChainFromPeerStrictV1AcceptsValidLongerChain(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := bc.GetLastBlock()

	difficulty := testExpectedNextDifficultyV1(
		t,
		bc,
	)

	next, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testReplacementCoinbaseTxV1(reward),
		},
		prev.Hash,
		"replacement-miner",
		difficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine replacement test block: %v",
			err,
		)
	}

	blocks := append(
		[]*Block(nil),
		bc.GetAllBlocks()...,
	)
	blocks = append(
		blocks,
		next,
	)

	if err := bc.ReplaceChainFromPeerStrictV1(
		blocks,
	); err != nil {
		t.Fatalf(
			"expected valid longer chain accepted, got: %v",
			err,
		)
	}

	if got := bc.GetBestHeight(); got != 1 {
		t.Fatalf(
			"expected height 1 after replacement, got %d",
			got,
		)
	}
}

func TestValidateReplacementChainStrictV1RejectsHashMismatch(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := bc.GetLastBlock()

	difficulty := testExpectedNextDifficultyV1(
		t,
		bc,
	)

	next, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testReplacementCoinbaseTxV1(reward),
		},
		prev.Hash,
		"replacement-miner",
		difficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine replacement test block: %v",
			err,
		)
	}

	next.Hash = []byte(
		"bad-hash",
	)

	blocks := append(
		[]*Block(nil),
		bc.GetAllBlocks()...,
	)
	blocks = append(
		blocks,
		next,
	)

	err = bc.ValidateReplacementChainStrictV1(
		blocks,
	)
	if err == nil {
		t.Fatal(
			"expected hash mismatch rejection",
		)
	}

	if !strings.Contains(
		err.Error(),
		"hash",
	) {
		t.Fatalf(
			"expected hash mismatch error, got: %v",
			err,
		)
	}
}

func TestValidateReplacementChainStrictV1RejectsUnexpectedDifficulty(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := bc.GetLastBlock()

	expectedDifficulty := testExpectedNextDifficultyV1(
		t,
		bc,
	)

	wrongDifficulty := expectedDifficulty + 1
	if wrongDifficulty > maxDifficultyBits {
		wrongDifficulty = expectedDifficulty - 1
	}

	next, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testReplacementCoinbaseTxV1(reward),
		},
		prev.Hash,
		"replacement-miner",
		wrongDifficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine wrong-difficulty replacement block: %v",
			err,
		)
	}

	blocks := append(
		[]*Block(nil),
		bc.GetAllBlocks()...,
	)
	blocks = append(
		blocks,
		next,
	)

	err = bc.ValidateReplacementChainStrictV1(
		blocks,
	)
	if err == nil {
		t.Fatal(
			"replacement chain with unexpected difficulty was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"difficulty mismatch",
	) {
		t.Fatalf(
			"expected difficulty mismatch error, got: %v",
			err,
		)
	}
}
