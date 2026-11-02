package token

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/miner"
)

func minerBridgeTestBlock(
	height int,
	previous []byte,
	minerAddress string,
	variant string,
) *blockchain.Block {
	hash := sha256.Sum256(
		[]byte(fmt.Sprintf(
			"qma0010:%s:%d:%x:%s",
			variant,
			height,
			previous,
			minerAddress,
		)),
	)

	return &blockchain.Block{
		Index:    height,
		PrevHash: append([]byte(nil), previous...),
		Hash:     append([]byte(nil), hash[:]...),
		Miner:    minerAddress,
	}
}

func minerBridgeTestChain(
	maxHeight int,
	minerAddress string,
	variant string,
) *blockchain.Blockchain {
	minerAddress = canonicalAddressFixture(
		minerAddress,
	)
	genesis := minerBridgeTestBlock(
		0,
		nil,
		"genesis",
		variant,
	)
	blocks := []*blockchain.Block{genesis}

	for height := 1; height <= maxHeight; height++ {
		blocks = append(
			blocks,
			minerBridgeTestBlock(
				height,
				blocks[len(blocks)-1].Hash,
				minerAddress,
				variant,
			),
		)
	}

	return &blockchain.Blockchain{
		Blocks: blocks,
	}
}

func appendCanonicalMinerBridgeTestBlock(
	chain *blockchain.Blockchain,
	maxHeight int,
	minerAddress string,
	variant string,
) {
	minerAddress = canonicalAddressFixture(
		minerAddress,
	)
	for chain.GetBestHeight() < maxHeight {
		height := chain.GetBestHeight() + 1
		previous := chain.GetLastBlock()
		chain.Blocks = append(
			chain.Blocks,
			minerBridgeTestBlock(
				height,
				previous.Hash,
				minerAddress,
				variant,
			),
		)
	}
}

func minerBridgeTestStatus(
	block *blockchain.Block,
) miner.MiningStatus {
	return miner.MiningStatus{
		BlockHeight: block.Index,
		BlockHash: append(
			[]byte(nil),
			block.Hash...,
		),
	}
}

func minerBridgeTestBridge(
	t *testing.T,
	chain *blockchain.Blockchain,
	directory string,
) *MinerTokenBridge {
	t.Helper()
	t.Setenv("QC_ENABLE_TOKEN_MINER_BRIDGE", "1")
	t.Setenv("QC_NODE_DIR", directory)

	bridge, err := NewMinerTokenBridge(
		MinerTokenConfig{
			Symbol:         "MINE",
			Name:           "Mineable Test Token",
			Decimals:       8,
			MaxSupply:      1_000_000,
			Owner:          canonicalAddressFixture("owner"),
			RewardPerBlock: 25,
			SaveDir:        directory,
		},
		chain,
	)
	if err != nil {
		t.Fatalf("new miner token bridge: %v", err)
	}

	return bridge
}

func TestMinerTokenBridgeV1RequiresCanonicalChain(t *testing.T) {
	t.Setenv("QC_ENABLE_TOKEN_MINER_BRIDGE", "1")

	_, err := NewMinerTokenBridge(
		MinerTokenConfig{
			Symbol:         "MINE",
			Owner:          canonicalAddressFixture("owner"),
			RewardPerBlock: 1,
		},
	)
	if !errors.Is(
		err,
		ErrMinerTokenCanonicalChainRequiredV1,
	) {
		t.Fatalf("canonical chain error = %v", err)
	}
}

func TestMinerTokenBridgeV1WaitsForSixCanonicalDescendants(
	t *testing.T,
) {
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	mined := chain.GetBlockByIndex(1)

	if err := bridge.processMinerTokenBlockV1(
		mined,
		minerBridgeTestStatus(mined),
	); err != nil {
		t.Fatalf("record pending block: %v", err)
	}
	if got := bridge.tok.TotalSupply(); got != 0 {
		t.Fatalf("supply before finality = %d, want 0", got)
	}

	appendCanonicalMinerBridgeTestBlock(
		chain,
		6,
		"miner",
		"canonical",
	)
	blockSix := chain.GetBlockByIndex(6)
	if err := bridge.processMinerTokenBlockV1(
		blockSix,
		minerBridgeTestStatus(blockSix),
	); err != nil {
		t.Fatalf("height six callback: %v", err)
	}
	if got := bridge.tok.TotalSupply(); got != 0 {
		t.Fatalf("supply with five descendants = %d, want 0", got)
	}

	appendCanonicalMinerBridgeTestBlock(
		chain,
		7,
		"miner",
		"canonical",
	)
	blockSeven := chain.GetBlockByIndex(7)
	if err := bridge.processMinerTokenBlockV1(
		blockSeven,
		minerBridgeTestStatus(blockSeven),
	); err != nil {
		t.Fatalf("height seven callback: %v", err)
	}
	if got := bridge.tok.TotalSupply(); got != 25 {
		t.Fatalf("supply after finality = %d, want 25", got)
	}
	if got := bridge.tok.BalanceOf(canonicalAddressFixture("miner")); got != 25 {
		t.Fatalf("miner balance = %d, want 25", got)
	}
}

func TestMinerTokenBridgeV1DuplicateCallbackMintsOnce(
	t *testing.T,
) {
	chain := minerBridgeTestChain(7, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	block := chain.GetBlockByIndex(1)
	status := minerBridgeTestStatus(block)

	for attempt := 0; attempt < 2; attempt++ {
		if err := bridge.processMinerTokenBlockV1(
			block,
			status,
		); err != nil {
			t.Fatalf("callback %d: %v", attempt, err)
		}
	}

	if got := bridge.tok.TotalSupply(); got != 25 {
		t.Fatalf("duplicate supply = %d, want 25", got)
	}
}

func TestMinerTokenBridgeV1RejectsStatusMismatch(
	t *testing.T,
) {
	chain := minerBridgeTestChain(1, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	block := chain.GetBlockByIndex(1)
	status := minerBridgeTestStatus(block)
	status.BlockHeight++

	err := bridge.processMinerTokenBlockV1(block, status)
	if !errors.Is(err, ErrMinerTokenBlockIdentityV1) {
		t.Fatalf("status mismatch error = %v", err)
	}
	if got := bridge.tok.TotalSupply(); got != 0 {
		t.Fatalf("mismatched status supply = %d", got)
	}
}

func TestMinerTokenBridgeV1DropsPreFinalityReorg(
	t *testing.T,
) {
	chain := minerBridgeTestChain(1, "miner", "original")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	original := chain.GetBlockByIndex(1)

	if err := bridge.processMinerTokenBlockV1(
		original,
		minerBridgeTestStatus(original),
	); err != nil {
		t.Fatalf("record original pending block: %v", err)
	}

	replacement := minerBridgeTestChain(7, "miner", "replacement")
	chain.Blocks = replacement.Blocks
	newHead := chain.GetBlockByIndex(7)

	if err := bridge.processMinerTokenBlockV1(
		newHead,
		minerBridgeTestStatus(newHead),
	); err != nil {
		t.Fatalf("replacement callback: %v", err)
	}
	if got := bridge.tok.TotalSupply(); got != 0 {
		t.Fatalf("reorged pending supply = %d, want 0", got)
	}

	records := bridge.tok.minerTokenBlocksSnapshotV1()
	if _, exists := records[minerTokenBlockKeyV1(1)]; exists {
		t.Fatal("reorged pending height was not removed")
	}
}

func TestMinerTokenBridgeV1FinalizedHeightConflictFailsClosed(
	t *testing.T,
) {
	chain := minerBridgeTestChain(7, "miner", "original")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	original := chain.GetBlockByIndex(1)

	if err := bridge.processMinerTokenBlockV1(
		original,
		minerBridgeTestStatus(original),
	); err != nil {
		t.Fatalf("finalize original block: %v", err)
	}

	replacement := minerBridgeTestChain(7, "miner", "replacement")
	chain.Blocks = replacement.Blocks
	conflict := chain.GetBlockByIndex(1)

	err := bridge.processMinerTokenBlockV1(
		conflict,
		minerBridgeTestStatus(conflict),
	)
	if !errors.Is(
		err,
		ErrMinerTokenFinalizedHeightConflictV1,
	) {
		t.Fatalf("finalized conflict error = %v", err)
	}
	if got := bridge.tok.TotalSupply(); got != 25 {
		t.Fatalf("conflict changed supply to %d", got)
	}
}

func TestMinerTokenBridgeV1SerializesDuplicateCallbacks(
	t *testing.T,
) {
	chain := minerBridgeTestChain(7, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, t.TempDir())
	block := chain.GetBlockByIndex(1)
	status := minerBridgeTestStatus(block)

	const workers = 8
	errorsCh := make(chan error, workers)
	start := make(chan struct{})

	for index := 0; index < workers; index++ {
		go func() {
			<-start
			errorsCh <- bridge.processMinerTokenBlockV1(
				block,
				status,
			)
		}()
	}

	close(start)
	for index := 0; index < workers; index++ {
		if err := <-errorsCh; err != nil {
			t.Fatalf("concurrent callback: %v", err)
		}
	}

	if got := bridge.tok.TotalSupply(); got != 25 {
		t.Fatalf("concurrent duplicate supply = %d", got)
	}
}
