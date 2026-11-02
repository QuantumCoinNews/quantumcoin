package blockchain

import (
	"encoding/hex"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestBlockchainPoWSelectionUsesNFTV4ActivationHeightV1(t *testing.T) {
	chain := NewBlockchain(1, 0)

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "quantumcoin-test-nft-pow",
			LegacyGenesisHex: hex.EncodeToString(chain.Blocks[0].Hash),
			ActivationHeight: 100,
			TargetProtocol:   qconsensus.ProtocolNFTV4,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}

	verified := verifiedActivationFixtureV1(t, policy)
	if err := chain.BindActivationPolicyV1(verified); err != nil {
		t.Fatalf("bind verified NFT V4 activation policy: %v", err)
	}

	cases := []struct {
		height   int
		protocol qconsensus.ProtocolVersion
	}{
		{height: 99, protocol: qconsensus.ProtocolLegacyV1},
		{height: 100, protocol: qconsensus.ProtocolNFTV4},
		{height: 101, protocol: qconsensus.ProtocolNFTV4},
	}

	for _, tc := range cases {
		if got := chain.ProtocolAtHeightV1(tc.height); got != tc.protocol {
			t.Fatalf("protocol at height %d = %q, want %q", tc.height, got, tc.protocol)
		}

		block := blockWithStoredPoWForProtocolV1(t, tc.protocol)
		block.Index = tc.height

		if err := chain.ValidateBlockPoWAtHeightV1(block); err != nil {
			t.Fatalf(
				"valid block rejected at height %d under %s: %v",
				tc.height,
				tc.protocol,
				err,
			)
		}
	}

	activatedBlock := blockWithStoredPoWForProtocolV1(t, qconsensus.ProtocolNFTV4)
	activatedBlock.Index = 100
	activatedBlock.NFTMintRecordsV1 = []NFTMintRecordV1{
		nftRecordSetFixtureV1(1, 0x44),
	}
	if err := chain.ValidateBlockPoWAtHeightV1(activatedBlock); err == nil {
		t.Fatal("NFT V4 block was accepted after its committed NFT record changed")
	}
}
