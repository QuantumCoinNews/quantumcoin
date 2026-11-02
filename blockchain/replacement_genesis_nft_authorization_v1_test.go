package blockchain

import (
	"strings"
	"testing"
)

func TestReplaceChainRejectsUnauthorizedGenesisNFTV1(t *testing.T) {
	chain := NewBlockchain(1, 0)
	previous := chain.GetLastBlock()

	reward, err := chain.ExpectedSubsidyUnitsForHeightV1(1)
	if err != nil {
		t.Fatalf("resolve baseline reward: %v", err)
	}

	next, err := NewBlock(
		1,
		[]*Transaction{testReplacementCoinbaseTxV1(reward)},
		previous.Hash,
		"replacement-miner",
		testExpectedNextDifficultyV1(t, chain),
	)
	if err != nil {
		t.Fatalf("build baseline replacement: %v", err)
	}

	incoming := cloneBlocksV1(chain.Blocks)
	incoming = append(incoming, next)

	if err := chain.ValidateReplacementChainStrictV1(incoming); err != nil {
		t.Fatalf("valid baseline replacement rejected: %v", err)
	}

	incoming[0].NFTMintRecordsV1 = []NFTMintRecordV1{
		nftRecordSetFixtureV1(1, 0x44),
	}

	err = chain.ReplaceChain(incoming)
	if err == nil {
		t.Fatal("QMA0068_RED: replacement accepted unauthorized NFT record injected into genesis")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "nft") {
		t.Fatalf("replacement failed outside NFT validation: %v", err)
	}
	if chain.GetBestHeight() != 0 ||
		len(chain.Blocks[0].NFTMintRecordsV1) != 0 {
		t.Fatal("rejected replacement changed local chain state")
	}
}
