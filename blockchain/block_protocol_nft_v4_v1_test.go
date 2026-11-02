package blockchain

import (
	"bytes"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestNewBlockAtProtocolV1UsesNFTV4PoW(t *testing.T) {
	previousHash := bytes.Repeat([]byte{0x11}, 32)
	record := nftRecordSetFixtureV1(1, 0x22)

	v4Block, err := NewBlockAtProtocolV1(
		100,
		nil,
		previousHash,
		"miner-test",
		0,
		[]NFTMintRecordV1{record},
		qconsensus.ProtocolNFTV4,
	)
	if err != nil {
		t.Fatalf("create NFT V4 block: %v", err)
	}
	if err := ValidateBlockPoWV4(v4Block); err != nil {
		t.Fatalf("new V4 block failed its own PoW validation: %v", err)
	}

	v4Block.NFTMintRecordsV1[0].RecipientPubKeyHash[0] ^= 0xff
	if err := ValidateBlockPoWV4(v4Block); err == nil {
		t.Fatal("V4 block remained valid after its NFT record changed")
	}

	legacyBlock, err := NewBlockAtProtocolV1(
		99,
		nil,
		previousHash,
		"miner-test",
		0,
		nil,
		qconsensus.ProtocolAtomV3,
	)
	if err != nil {
		t.Fatalf("create legacy protocol block: %v", err)
	}
	if err := ValidateBlockPoWV1(legacyBlock); err != nil {
		t.Fatalf("legacy block failed V1 PoW validation: %v", err)
	}

	if _, err := NewBlockAtProtocolV1(
		99,
		nil,
		previousHash,
		"miner-test",
		0,
		[]NFTMintRecordV1{record},
		qconsensus.ProtocolAtomV3,
	); err == nil {
		t.Fatal("legacy protocol created a block with uncommitted NFT records")
	}
}
