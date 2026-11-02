package blockchain

import "testing"

func TestCloneBlockV1DeepCopiesNFTMintRecords(t *testing.T) {
	original := &Block{
		NFTMintRecordsV1: []NFTMintRecordV1{
			nftRecordSetFixtureV1(1, 0x11),
		},
	}

	cloned := cloneBlockV1(original)
	if cloned == nil {
		t.Fatal("cloneBlockV1 returned nil")
	}
	if len(cloned.NFTMintRecordsV1) != 1 {
		t.Fatalf(
			"cloned NFT record count = %d, want 1",
			len(cloned.NFTMintRecordsV1),
		)
	}

	cloned.NFTMintRecordsV1[0].Serial = 2
	cloned.NFTMintRecordsV1[0].RecipientPubKeyHash[0] = 0x77
	cloned.NFTMintRecordsV1[0].MetadataHash[0] = 0x88

	if original.NFTMintRecordsV1[0].Serial != 1 {
		t.Fatal("mutating cloned NFT serial changed the original block")
	}
	if original.NFTMintRecordsV1[0].RecipientPubKeyHash[0] != 0x11 {
		t.Fatal("mutating cloned NFT owner hash changed the original block")
	}
	if original.NFTMintRecordsV1[0].MetadataHash[0] == 0x88 {
		t.Fatal("mutating cloned NFT metadata hash changed the original block")
	}

	original.NFTMintRecordsV1[0].RecipientPubKeyHash[1] = 0x99
	if cloned.NFTMintRecordsV1[0].RecipientPubKeyHash[1] == 0x99 {
		t.Fatal("mutating original NFT owner hash changed the cloned block")
	}
}
