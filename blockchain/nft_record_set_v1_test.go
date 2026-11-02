package blockchain

import "testing"

func nftRecordSetFixtureV1(serial uint32, owner byte) NFTMintRecordV1 {
	metadata := [32]byte{}
	metadata[0] = byte(serial)

	recipient := make([]byte, 20)
	for i := range recipient {
		recipient[i] = owner
	}

	return NFTMintRecordV1{
		NetworkID:           "quantumcoin-test",
		CampaignID:          "telegram-monthly",
		AwardPeriod:         "2026-09",
		Serial:              serial,
		RecipientPubKeyHash: recipient,
		MetadataHash:        metadata,
	}
}

func TestNFTMintRecordsDigestV1BindsOrderedRecords(t *testing.T) {
	firstRecord := nftRecordSetFixtureV1(1, 0x11)
	secondRecord := nftRecordSetFixtureV1(2, 0x22)

	first, err := NFTMintRecordsDigestV1([]NFTMintRecordV1{
		firstRecord,
		secondRecord,
	})
	if err != nil {
		t.Fatalf("digest records: %v", err)
	}

	repeated, err := NFTMintRecordsDigestV1([]NFTMintRecordV1{
		firstRecord,
		secondRecord,
	})
	if err != nil {
		t.Fatalf("digest repeated records: %v", err)
	}
	if first != repeated {
		t.Fatal("same NFT record list produced different digests")
	}

	reordered, err := NFTMintRecordsDigestV1([]NFTMintRecordV1{
		secondRecord,
		firstRecord,
	})
	if err != nil {
		t.Fatalf("digest reordered records: %v", err)
	}
	if first == reordered {
		t.Fatal("reordering NFT records did not change the digest")
	}

	changedRecord := nftRecordSetFixtureV1(2, 0x33)
	changed, err := NFTMintRecordsDigestV1([]NFTMintRecordV1{
		firstRecord,
		changedRecord,
	})
	if err != nil {
		t.Fatalf("digest changed records: %v", err)
	}
	if first == changed {
		t.Fatal("changing NFT record ownership did not change the digest")
	}
}

func TestNFTMintRecordsDigestV1RejectsInvalidRecord(t *testing.T) {
	invalid := nftRecordSetFixtureV1(1, 0x11)
	invalid.RecipientPubKeyHash = []byte{1}

	if _, err := NFTMintRecordsDigestV1([]NFTMintRecordV1{invalid}); err == nil {
		t.Fatal("invalid NFT mint record was accepted")
	}
}
