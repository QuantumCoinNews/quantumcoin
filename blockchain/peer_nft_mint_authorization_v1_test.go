package blockchain

import (
	"encoding/hex"
	"errors"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestPeerRejectsUnauthenticatedNFTMintRecordV1(t *testing.T) {
	chain := NewBlockchain(1, 0)
	genesisHex := hex.EncodeToString(chain.Blocks[0].Hash)

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "quantumcoin-peer-nft-auth-test-v1",
			LegacyGenesisHex: genesisHex,
			ActivationHeight: 1,
			TargetProtocol:   qconsensus.ProtocolNFTV4,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}
	if err := chain.BindActivationPolicyV1(
		verifiedActivationFixtureV1(t, policy),
	); err != nil {
		t.Fatalf("bind verified NFT V4 activation policy: %v", err)
	}

	prev := chain.Blocks[len(chain.Blocks)-1]
	reward, err := chain.ExpectedSubsidyUnitsForHeightV1(prev.Index + 1)
	if err != nil {
		t.Fatalf("resolve NFT V4 subsidy units: %v", err)
	}
	if reward <= 0 {
		t.Fatalf("NFT V4 subsidy units must be positive, got %d", reward)
	}
	difficulty := testExpectedNextDifficultyV1(t, chain)

	makeBlock := func(records []NFTMintRecordV1) *Block {
		t.Helper()
		block, err := NewBlockAtProtocolV1(
			prev.Index+1,
			[]*Transaction{testPeerCoinbaseTxV2(reward)},
			prev.Hash,
			"peer-miner",
			difficulty,
			records,
			qconsensus.ProtocolNFTV4,
		)
		if err != nil {
			t.Fatalf("build NFT V4 peer block: %v", err)
		}
		return block
	}

	emptyBlock := makeBlock(nil)
	if err := chain.ValidatePeerBlockStrictV2(emptyBlock); err != nil {
		t.Fatalf("valid empty-record NFT V4 baseline was rejected: %v", err)
	}

	record := nftRecordSetFixtureV1(1, 0x44)
	record.NetworkID = policy.Plan.NetworkID
	candidate := makeBlock([]NFTMintRecordV1{record})

	err = chain.ValidatePeerBlockStrictV2(candidate)
	if err == nil {
		t.Fatal("unauthorized NFT record accepted: missing award authorization")
	}
	if !errors.Is(err, ErrNFTAwardTrustUnboundV1) {
		t.Fatalf("candidate did not fail on missing local NFT authority trust: %v", err)
	}
}
