package blockchain

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"

	qconsensus "quantumcoin/consensus"
)

func testPeerCoinbaseTxV2(amount int) *Transaction {
	tx := &Transaction{
		ID:     nil,
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount:     amount,
				PubKeyHash: []byte("peer-miner"),
			},
		},
		Timestamp: time.Now(),
		Sender:    "COINBASE",
		Amount:    float64(amount),
	}

	tx.ID = tx.Hash()

	return tx
}

func testExpectedNextDifficultyV1(
	t *testing.T,
	bc *Blockchain,
) int {
	t.Helper()

	difficulty, err := bc.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatalf(
			"resolve expected test difficulty: %v",
			err,
		)
	}

	return difficulty
}

func TestValidatePeerBlockStrictV2RejectsHashMismatch(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := bc.Blocks[len(bc.Blocks)-1]

	blk, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testPeerCoinbaseTxV2(reward),
		},
		prev.Hash,
		"peer-miner",
		testExpectedNextDifficultyV1(t, bc),
	)
	if err != nil {
		t.Fatalf(
			"mine peer test block: %v",
			err,
		)
	}

	blk.Hash = []byte(
		"not-the-real-pow-hash",
	)

	err = bc.ValidatePeerBlockStrictV2(
		blk,
	)
	if err == nil {
		t.Fatal(
			"expected proof-of-work validation rejection",
		)
	}

	if !strings.Contains(
		err.Error(),
		"proof-of-work validation failed",
	) {
		t.Fatalf(
			"expected proof-of-work validation error, got: %v",
			err,
		)
	}
}

func TestAddBlockFromPeerStrictV2AcceptsValidNextBlock(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := bc.Blocks[len(bc.Blocks)-1]

	blk, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testPeerCoinbaseTxV2(reward),
		},
		prev.Hash,
		"peer-miner",
		testExpectedNextDifficultyV1(t, bc),
	)
	if err != nil {
		t.Fatalf(
			"mine peer test block: %v",
			err,
		)
	}

	if err := bc.AddBlockFromPeerStrictV2(
		blk,
	); err != nil {
		t.Fatalf(
			"expected valid peer block to be accepted, got: %v",
			err,
		)
	}

	if got := bc.GetBestHeight(); got != 1 {
		t.Fatalf(
			"expected height 1 after peer block, got %d",
			got,
		)
	}
}

func TestValidatePeerBlockStrictV2RejectsWrongIndex(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := bc.Blocks[len(bc.Blocks)-1]

	blk, err := NewBlock(
		prev.Index+2,
		[]*Transaction{
			testPeerCoinbaseTxV2(reward),
		},
		prev.Hash,
		"peer-miner",
		testExpectedNextDifficultyV1(t, bc),
	)
	if err != nil {
		t.Fatalf(
			"mine peer test block: %v",
			err,
		)
	}

	err = bc.ValidatePeerBlockStrictV2(
		blk,
	)
	if err == nil {
		t.Fatal(
			"expected wrong index rejection",
		)
	}

	if !strings.Contains(
		err.Error(),
		"bad index sequence",
	) {
		t.Fatalf(
			"expected bad index sequence error, got: %v",
			err,
		)
	}
}

func TestValidatePeerBlockStrictV2UsesLegacyProtocolWhenPolicyDisabled(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	if err := bc.BindCanonicalActivationPolicyV1(); err != nil {
		t.Fatalf(
			"bind canonical activation policy: %v",
			err,
		)
	}

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := bc.Blocks[len(bc.Blocks)-1]

	blk, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testPeerCoinbaseTxV2(reward),
		},
		prev.Hash,
		"peer-miner",
		testExpectedNextDifficultyV1(t, bc),
	)
	if err != nil {
		t.Fatalf(
			"mine peer test block: %v",
			err,
		)
	}

	if err := bc.ValidatePeerBlockStrictV2(
		blk,
	); err != nil {
		t.Fatalf(
			"disabled canonical policy should preserve legacy validation: %v",
			err,
		)
	}
}

func TestValidatePeerBlockStrictV2RejectsContinuityV2AtActivationHeight(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	genesisHex := hex.EncodeToString(
		bc.Blocks[0].Hash,
	)

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "quantumcoin-peer-validation-test-v1",
			LegacyGenesisHex: genesisHex,
			ActivationHeight: 1,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}

	if err := bc.BindActivationPolicyV1(
		verifiedActivationFixtureV1(t, policy),
	); err != nil {
		t.Fatalf(
			"bind activation policy: %v",
			err,
		)
	}

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := bc.Blocks[len(bc.Blocks)-1]

	blk, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testPeerCoinbaseTxV2(reward),
		},
		prev.Hash,
		"peer-miner",
		testExpectedNextDifficultyV1(t, bc),
	)
	if err != nil {
		t.Fatalf(
			"mine peer test block: %v",
			err,
		)
	}

	err = bc.ValidatePeerBlockStrictV2(
		blk,
	)
	if err == nil {
		t.Fatal(
			"continuity v2 block was accepted before implementation",
		)
	}

	if !strings.Contains(
		err.Error(),
		"not implemented",
	) {
		t.Fatalf(
			"expected continuity v2 rejection, got: %v",
			err,
		)
	}
}

func TestValidatePeerBlockStrictV2RejectsUnexpectedDifficulty(
	t *testing.T,
) {
	bc := NewBlockchain(1, 0)

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	prev := bc.Blocks[len(bc.Blocks)-1]
	expectedDifficulty := testExpectedNextDifficultyV1(
		t,
		bc,
	)

	wrongDifficulty := expectedDifficulty + 1
	if wrongDifficulty > maxDifficultyBits {
		wrongDifficulty = expectedDifficulty - 1
	}

	blk, err := NewBlock(
		prev.Index+1,
		[]*Transaction{
			testPeerCoinbaseTxV2(reward),
		},
		prev.Hash,
		"peer-miner",
		wrongDifficulty,
	)
	if err != nil {
		t.Fatalf(
			"mine wrong-difficulty peer test block: %v",
			err,
		)
	}

	err = bc.ValidatePeerBlockStrictV2(
		blk,
	)
	if err == nil {
		t.Fatal(
			"peer block with unexpected difficulty was accepted",
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
