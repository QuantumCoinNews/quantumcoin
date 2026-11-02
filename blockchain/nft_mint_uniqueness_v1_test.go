package blockchain

import (
	"strings"
	"testing"
)

func TestNFTMintUniquenessOneAwardPerNetworkPeriodV1(t *testing.T) {
	existingRecord := nftRecordSetFixtureV1(1, 0x11)
	existingBlock := &Block{
		Index:            1,
		NFTMintRecordsV1: []NFTMintRecordV1{existingRecord},
	}

	otherCampaignSamePeriod := nftRecordSetFixtureV1(2, 0x22)
	otherCampaignSamePeriod.CampaignID = "second-campaign"

	err := validateNFTMintUniquenessV1(
		[]*Block{existingBlock},
		&Block{
			Index:            2,
			NFTMintRecordsV1: []NFTMintRecordV1{otherCampaignSamePeriod},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "duplicate NFT award period") {
		t.Fatalf("same network and award period were not rejected: %v", err)
	}

	first := nftRecordSetFixtureV1(3, 0x33)
	second := nftRecordSetFixtureV1(4, 0x44)
	second.CampaignID = "second-campaign"

	err = validateNFTMintUniquenessV1(
		nil,
		&Block{
			Index:            1,
			NFTMintRecordsV1: []NFTMintRecordV1{first, second},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "duplicate NFT award period") {
		t.Fatalf("two awards in one block and period were not rejected: %v", err)
	}

	nextPeriod := nftRecordSetFixtureV1(5, 0x55)
	nextPeriod.AwardPeriod = "2026-10"

	if err := validateNFTMintUniquenessV1(
		[]*Block{existingBlock},
		&Block{
			Index:            2,
			NFTMintRecordsV1: []NFTMintRecordV1{nextPeriod},
		},
	); err != nil {
		t.Fatalf("different award period was rejected: %v", err)
	}
}
