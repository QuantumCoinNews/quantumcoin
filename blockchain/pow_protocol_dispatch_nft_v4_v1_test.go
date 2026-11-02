package blockchain

import (
	"bytes"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func blockWithStoredPoWForProtocolV1(
	t *testing.T,
	protocol qconsensus.ProtocolVersion,
) *Block {
	t.Helper()

	block := &Block{
		Index:            100,
		Timestamp:        1700,
		PrevHash:         bytes.Repeat([]byte{0x11}, 32),
		Difficulty:       0,
		Nonce:            9,
		NFTMintRecordsV1: []NFTMintRecordV1{nftRecordSetFixtureV1(1, 0x11)},
	}

	var hash [32]byte
	var err error

	if protocol == qconsensus.ProtocolNFTV4 {
		header, headerErr := canonicalPoWHeaderV4(block)
		if headerErr != nil {
			t.Fatalf("build V4 header: %v", headerErr)
		}
		hash, err = qconsensus.ComputePoWHashV4(header, uint64(block.Nonce))
	} else {
		header, headerErr := canonicalPoWHeaderV1(block)
		if headerErr != nil {
			t.Fatalf("build V1 header: %v", headerErr)
		}
		hash, err = qconsensus.ComputePoWHashV1(header, uint64(block.Nonce))
	}
	if err != nil {
		t.Fatalf("compute stored PoW: %v", err)
	}

	block.Hash = append([]byte(nil), hash[:]...)
	return block
}

func TestValidateBlockPoWAtProtocolV1SelectsNFTV4Commitment(t *testing.T) {
	v4Block := blockWithStoredPoWForProtocolV1(t, qconsensus.ProtocolNFTV4)

	if err := ValidateBlockPoWAtProtocolV1(v4Block, qconsensus.ProtocolNFTV4); err != nil {
		t.Fatalf("valid V4 block rejected: %v", err)
	}

	v4Block.NFTMintRecordsV1 = []NFTMintRecordV1{
		nftRecordSetFixtureV1(1, 0x22),
	}
	if err := ValidateBlockPoWAtProtocolV1(v4Block, qconsensus.ProtocolNFTV4); err == nil {
		t.Fatal("V4 PoW accepted after the committed NFT record changed")
	}

	legacyBlock := blockWithStoredPoWForProtocolV1(t, qconsensus.ProtocolAtomV3)
	if err := ValidateBlockPoWAtProtocolV1(legacyBlock, qconsensus.ProtocolAtomV3); err != nil {
		t.Fatalf("valid legacy V1 PoW rejected: %v", err)
	}

	legacyBlock.NFTMintRecordsV1 = []NFTMintRecordV1{
		nftRecordSetFixtureV1(1, 0x33),
	}
	if err := ValidateBlockPoWAtProtocolV1(legacyBlock, qconsensus.ProtocolAtomV3); err != nil {
		t.Fatalf("legacy V1 PoW changed with NFT record: %v", err)
	}
}
