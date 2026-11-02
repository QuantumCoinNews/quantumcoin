package blockchain

import (
	"errors"
	"reflect"
	"testing"
)

func TestNFTMintDoesNotMutateTipMetadataV1(t *testing.T) {
	t.Setenv("QC_ENABLE_CHAIN_NFT_MINT", "1")

	chain := NewBlockchain(1, 0)
	if chain == nil || len(chain.Blocks) == 0 {
		t.Fatal("NewBlockchain did not provide a tip block")
	}

	tip := chain.Blocks[len(chain.Blocks)-1]
	tip.Metadata["existing"] = "preserve-me"
	before := make(map[string]string, len(tip.Metadata))
	for key, value := range tip.Metadata {
		before[key] = value
	}

	id, err := chain.MintNFT("QC_TEST_RECIPIENT", "monthly_rare", map[string]string{
		"trait": "test",
	})

	if id != "" || !errors.Is(err, ErrNFTMintDisabled) {
		t.Fatalf("mint returned id=%q, err=%v; want empty ID and ErrNFTMintDisabled", id, err)
	}
	if !reflect.DeepEqual(tip.Metadata, before) {
		t.Fatalf("MintNFT changed tip metadata: before=%v after=%v", before, tip.Metadata)
	}
}
