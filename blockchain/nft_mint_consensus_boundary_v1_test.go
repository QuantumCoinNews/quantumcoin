package blockchain

import (
	"errors"
	"testing"
)

func TestNFTMintFlagCannotBypassConsensusBoundaryV1(t *testing.T) {
	t.Setenv("QC_ENABLE_CHAIN_NFT_MINT", "1")

	var chain *Blockchain
	id, err := chain.MintNFT("QC_TEST_RECIPIENT", "monthly_rare", nil)

	if id != "" || !errors.Is(err, ErrNFTMintDisabled) {
		t.Fatalf("mint returned id=%q, err=%v; want empty ID and ErrNFTMintDisabled", id, err)
	}
}
