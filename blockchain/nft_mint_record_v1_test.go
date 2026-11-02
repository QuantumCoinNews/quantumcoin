package blockchain

import (
	"bytes"
	"testing"
)

func TestNFTMintRecordV1CanonicalIdentity(t *testing.T) {
	record := NFTMintRecordV1{
		NetworkID:           "quantumcoin-testnet",
		CampaignID:          "QC_TELEGRAM_RARE_NFT_V1",
		AwardPeriod:         "2026-10",
		Serial:              1,
		RecipientPubKeyHash: bytes.Repeat([]byte{0x11}, 20),
		MetadataHash:        [32]byte{0x7a, 0x01},
	}

	firstID, err := NFTTokenIDV1(record)
	if err != nil {
		t.Fatalf("first NFT token ID: %v", err)
	}
	secondID, err := NFTTokenIDV1(record)
	if err != nil {
		t.Fatalf("second NFT token ID: %v", err)
	}
	if firstID != secondID {
		t.Fatal("same NFT mint record produced different token IDs")
	}

	firstDigest, err := NFTMintRecordDigestV1(record)
	if err != nil {
		t.Fatalf("first mint record digest: %v", err)
	}

	transferred := record
	transferred.RecipientPubKeyHash = bytes.Repeat([]byte{0x22}, 20)

	transferredID, err := NFTTokenIDV1(transferred)
	if err != nil {
		t.Fatalf("transferred NFT token ID: %v", err)
	}
	if firstID != transferredID {
		t.Fatal("recipient change altered immutable NFT token ID")
	}

	transferredDigest, err := NFTMintRecordDigestV1(transferred)
	if err != nil {
		t.Fatalf("transferred mint record digest: %v", err)
	}
	if firstDigest == transferredDigest {
		t.Fatal("mint record digest did not bind the initial recipient")
	}

	changedAsset := record
	changedAsset.MetadataHash[0] ^= 0xff
	changedAssetID, err := NFTTokenIDV1(changedAsset)
	if err != nil {
		t.Fatalf("changed asset NFT token ID: %v", err)
	}
	if firstID == changedAssetID {
		t.Fatal("metadata hash change did not alter NFT token ID")
	}
}
