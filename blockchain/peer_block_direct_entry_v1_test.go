package blockchain

import (
	"strings"
	"testing"
)

func testValidPeerBlockForDirectEntryV1(
	t *testing.T,
	chain *Blockchain,
	index int,
) *Block {
	t.Helper()

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	previous := chain.Blocks[len(chain.Blocks)-1]

	block, err := NewBlock(
		index,
		[]*Transaction{
			testPeerCoinbaseTxV2(reward),
		},
		previous.Hash,
		"peer-miner",
		1,
	)
	if err != nil {
		t.Fatalf("mine direct peer test block: %v", err)
	}

	return block
}

func TestAddBlockFromPeerDirectRejectsWrongIndexV1(t *testing.T) {
	chain := NewBlockchain(1, 0)
	previous := chain.Blocks[len(chain.Blocks)-1]

	block := testValidPeerBlockForDirectEntryV1(
		t,
		chain,
		previous.Index+2,
	)

	err := chain.AddBlockFromPeer(block)
	if err == nil {
		t.Fatal("direct peer entry accepted wrong block index")
	}

	if !strings.Contains(err.Error(), "bad index sequence") {
		t.Fatalf(
			"expected bad index sequence error, got: %v",
			err,
		)
	}

	if got := chain.GetBestHeight(); got != 0 {
		t.Fatalf(
			"rejected peer block changed chain height: got %d, want 0",
			got,
		)
	}
}

func TestAddBlockFromPeerDirectRejectsNilBlockchainV1(t *testing.T) {
	source := NewBlockchain(1, 0)
	previous := source.Blocks[len(source.Blocks)-1]

	block := testValidPeerBlockForDirectEntryV1(
		t,
		source,
		previous.Index+1,
	)

	var chain *Blockchain

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf(
				"direct peer entry panicked for nil blockchain: %v",
				recovered,
			)
		}
	}()

	err := chain.AddBlockFromPeer(block)
	if err == nil {
		t.Fatal("nil blockchain accepted peer block")
	}
}
