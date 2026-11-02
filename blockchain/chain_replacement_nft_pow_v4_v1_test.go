package blockchain

import (
	"encoding/hex"
	"strings"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestReplaceChainPassesNFTV4PoWBeforeTransactionValidationV1(t *testing.T) {
	chain := NewBlockchain(1, 0)
	genesis := chain.Blocks[0]

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "quantumcoin-test-reorg-v4",
			LegacyGenesisHex: hex.EncodeToString(genesis.Hash),
			ActivationHeight: 1,
			TargetProtocol:   qconsensus.ProtocolNFTV4,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}
	if err := chain.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf("bind verified activation policy: %v", err)
	}

	// Valid V4 PoW/basic layout, with an excessive canonical coinbase.
	// The strict writer now checks basic layout before canonical consensus;
	// an empty-transaction synthetic fixture cannot test PoW dispatch there.
	reward, err := chain.ExpectedSubsidyUnitsForHeightV1(1)
	if err != nil {
		t.Fatal(err)
	}
	block, err := NewBlockAtProtocolV1(1,
		[]*Transaction{testPeerCoinbaseTxV2(reward + 1)}, genesis.Hash,
		"test-only-replacement-pow", testExpectedNextDifficultyV1(t, chain),
		nil, qconsensus.ProtocolNFTV4)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBlockBasic(block); err != nil {
		t.Fatal(err)
	}
	if err := chain.ValidateBlockPoWAtHeightV1(block); err != nil {
		t.Fatal(err)
	}
	err = chain.replaceChainAfterStrictV1([]*Block{genesis, block})
	if err == nil {
		t.Fatal("replacement with excessive coinbase should fail canonical reward validation")
	}
	if !strings.Contains(err.Error(), "coinbase mismatch") {
		t.Fatalf("valid V4 PoW did not reach canonical coinbase validation: %v", err)
	}
}
