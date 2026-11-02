package nft

import (
	"errors"
	"testing"
)

func TestMinerNFTLegacyPathFailsClosedV1(t *testing.T) {
	id, err := CheckAndMintMinerNFT("QC_TEST_MINER", 10)
	if id != "" {
		t.Fatalf("disabled miner NFT path returned ID %q", id)
	}
	if !errors.Is(err, ErrNFTAutoMintDisabled) {
		t.Fatalf("disabled miner NFT path returned error %v; want ErrNFTAutoMintDisabled", err)
	}
}
