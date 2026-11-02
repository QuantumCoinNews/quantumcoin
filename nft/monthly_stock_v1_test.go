package nft

import (
	"encoding/json"
	"os"
	"testing"
)

type monthlyNFTStockTestFile struct {
	Version        string `json:"version"`
	Year           int    `json:"year"`
	TotalPlanned   int    `json:"totalPlanned"`
	AwardsPerMonth int    `json:"awardsPerMonth"`
	Rarity         string `json:"rarity"`
	Items          []struct {
		StockNo             int    `json:"stockNo"`
		ID                  string `json:"id"`
		Period              string `json:"period"`
		Name                string `json:"name"`
		Rarity              string `json:"rarity"`
		StockStatus         string `json:"stockStatus"`
		ArtworkStatus       string `json:"artworkStatus"`
		MetadataStatus      string `json:"metadataStatus"`
		FinalReady          bool   `json:"finalReady"`
		MainnetMintStatus   string `json:"mainnetMintStatus"`
		ImagePath           string `json:"imagePath"`
		MetadataPath        string `json:"metadataPath"`
		AIReviewStatus      string `json:"aiReviewStatus"`
		AdminApprovalStatus string `json:"adminApprovalStatus"`
	} `json:"items"`
}

func TestMonthlyRareNFTStock2026File(t *testing.T) {
	raw, err := os.ReadFile("catalog/monthly_rare_nft_stock_2026.json")
	if err != nil {
		t.Fatalf("stock file missing: %v", err)
	}

	var stock monthlyNFTStockTestFile
	if err := json.Unmarshal(raw, &stock); err != nil {
		t.Fatalf("stock json invalid: %v", err)
	}

	if stock.Version != "NFT_MONTHLY_STOCK_PRE_MAINNET_V1" {
		t.Fatalf("bad stock version: %s", stock.Version)
	}
	if stock.Year != 2026 {
		t.Fatalf("bad year: %d", stock.Year)
	}
	if stock.TotalPlanned != 12 || len(stock.Items) != 12 {
		t.Fatalf("expected 12 stock items, got planned=%d len=%d", stock.TotalPlanned, len(stock.Items))
	}
	if stock.AwardsPerMonth != 1 {
		t.Fatalf("awards per month must be 1")
	}
	if stock.Rarity != "rare" {
		t.Fatalf("stock rarity must be rare")
	}

	seenID := map[string]bool{}
	seenPeriod := map[string]bool{}
	seenName := map[string]bool{}

	for _, item := range stock.Items {
		if item.StockNo <= 0 || item.StockNo > 12 {
			t.Fatalf("bad stock number: %+v", item)
		}
		if item.ID == "" || item.Period == "" || item.Name == "" {
			t.Fatalf("missing id/period/name: %+v", item)
		}
		if item.Rarity != "rare" {
			t.Fatalf("all stock NFTs must be rare: %+v", item)
		}
		if item.FinalReady {
			t.Fatalf("pre-mainnet stock must not be final ready yet: %+v", item)
		}
		if item.MainnetMintStatus != "NOT_MINTED" {
			t.Fatalf("pre-mainnet stock must not be minted: %+v", item)
		}
		if item.ImagePath == "" || item.MetadataPath == "" {
			t.Fatalf("image/metadata paths required: %+v", item)
		}
		if item.AIReviewStatus == "" || item.AdminApprovalStatus == "" {
			t.Fatalf("review statuses required: %+v", item)
		}
		if seenID[item.ID] || seenPeriod[item.Period] || seenName[item.Name] {
			t.Fatalf("stock items must be unique: %+v", item)
		}
		seenID[item.ID] = true
		seenPeriod[item.Period] = true
		seenName[item.Name] = true
	}
}
