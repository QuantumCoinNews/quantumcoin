package blockchain

import (
	"bytes"
	"testing"
)

func TestCanonicalPoWHeaderV4UsesBlockNFTRecordsV1(t *testing.T) {
	block := &Block{
		PrevHash:   bytes.Repeat([]byte{0x11}, 32),
		Timestamp:  1700,
		Difficulty: 8,
		NFTMintRecordsV1: []NFTMintRecordV1{
			nftRecordSetFixtureV1(1, 0x11),
		},
	}

	headerA, err := canonicalPoWHeaderV4(block)
	if err != nil {
		t.Fatalf("build V4 header: %v", err)
	}

	wantDigestA, err := NFTMintRecordsDigestV1(block.NFTMintRecordsV1)
	if err != nil {
		t.Fatalf("digest block NFT records: %v", err)
	}
	if !bytes.Equal(headerA.NFTRecordDigest, wantDigestA[:]) {
		t.Fatal("V4 header NFT digest does not match the block records")
	}

	legacyA, err := SerializePoWHeaderV1(block, 9)
	if err != nil {
		t.Fatalf("serialize legacy header A: %v", err)
	}

	block.NFTMintRecordsV1 = []NFTMintRecordV1{
		nftRecordSetFixtureV1(1, 0x22),
	}

	headerB, err := canonicalPoWHeaderV4(block)
	if err != nil {
		t.Fatalf("build changed V4 header: %v", err)
	}
	if bytes.Equal(headerA.NFTRecordDigest, headerB.NFTRecordDigest) {
		t.Fatal("changing a block NFT record did not change the V4 header digest")
	}

	legacyB, err := SerializePoWHeaderV1(block, 9)
	if err != nil {
		t.Fatalf("serialize legacy header B: %v", err)
	}
	if !bytes.Equal(legacyA, legacyB) {
		t.Fatal("adding or changing NFT records changed legacy V1 PoW bytes")
	}
}
