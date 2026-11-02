package nft

import (
	"errors"
	"testing"
)

func TestMinerNFTLegacyPathDisabledForAllInputsV1(t *testing.T) {
	t.Setenv("QC_ENABLE_CHAIN_NFT_MINT", "1")
	t.Setenv("QC_ENABLE_MINER_NFT_REWARD", "1")

	cases := []struct {
		name    string
		address string
		block   int
	}{
		{"empty_address", "", 0},
		{"normal_address", "QC_TEST_MINER", 1},
		{"negative_block", "QC_TEST_MINER", -1},
		{"large_block", "QC_TEST_MINER", 1_000_000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := CheckAndMintMinerNFT(tc.address, tc.block)
			if id != "" {
				t.Fatalf("disabled legacy path returned NFT ID %q", id)
			}
			if !errors.Is(err, ErrNFTAutoMintDisabled) {
				t.Fatalf("error = %v; want ErrNFTAutoMintDisabled", err)
			}
		})
	}
}
