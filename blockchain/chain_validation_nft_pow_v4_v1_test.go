package blockchain

import (
	"encoding/hex"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func resealNFTV4BlockPoWForTestV1(t *testing.T, block *Block) {
	t.Helper()

	header, err := canonicalPoWHeaderV4(block)
	if err != nil {
		t.Fatalf("build V4 header: %v", err)
	}

	hash, err := qconsensus.ComputePoWHashV4(header, uint64(block.Nonce))
	if err != nil {
		t.Fatalf("compute V4 hash: %v", err)
	}
	block.Hash = append([]byte(nil), hash[:]...)
}

func TestIsValidChainUsesNFTV4PoWAtActivationHeightV1(t *testing.T) {
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
	if err := chain.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf("bind verified activation policy: %v", err)
	}

	block := blockWithStoredPoWForProtocolV1(t, qconsensus.ProtocolNFTV4)
	block.Index = 1
	block.PrevHash = append([]byte(nil), genesis.Hash...)
	block.NFTMintRecordsV1 = nil
	resealNFTV4BlockPoWForTestV1(t, block)
	chain.Blocks = []*Block{genesis, block}

	if !chain.IsValidChain() {
		t.Fatal("IsValidChain rejected a valid empty-record NFT V4 block")
	}

	record := nftRecordSetFixtureV1(1, 0x55)
	record.NetworkID = policy.Plan.NetworkID
	block.NFTMintRecordsV1 = []NFTMintRecordV1{record}
	resealNFTV4BlockPoWForTestV1(t, block)

	if chain.IsValidChain() {
		t.Fatal("IsValidChain accepted an NFT mint record without award authorization")
	}
}
