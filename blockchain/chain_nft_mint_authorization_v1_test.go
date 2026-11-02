package blockchain

import (
	"encoding/hex"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestIsValidChainRejectsUnauthenticatedNFTRecordV1(t *testing.T) {
	chain := NewBlockchain(1, 0)
	genesis := chain.Blocks[0]

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "quantumcoin-test-chain-v4",
			LegacyGenesisHex: hex.EncodeToString(genesis.Hash),
			ActivationHeight: 1,
			TargetProtocol:   qconsensus.ProtocolNFTV4,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}
	if err := chain.BindActivationPolicyV1(
		verifiedActivationFixtureV1(t, policy),
	); err != nil {
		t.Fatalf("bind NFT V4 activation policy: %v", err)
	}

	block := blockWithStoredPoWForProtocolV1(
		t,
		qconsensus.ProtocolNFTV4,
	)
	block.Index = 1
	block.PrevHash = append([]byte(nil), genesis.Hash...)
	block.NFTMintRecordsV1[0].NetworkID = policy.Plan.NetworkID
	resealNFTV4BlockPoWForTestV1(t, block)

	chain.Blocks = []*Block{genesis, block}

	if chain.IsValidChain() {
		t.Fatal("IsValidChain accepted an NFT mint record without award authorization")
	}
}
