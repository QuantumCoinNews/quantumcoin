package nft

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// The checked-in stock is the sole source of monthly names, concepts and paths.
// BuildMonthlyRareNFTPlaceholderPlanV1 always keeps its mint flags disabled,
// independently of any later preparation changes in this stock.
//
//go:embed catalog/monthly_rare_nft_stock_2026.json
var monthlyNFTStock2026SourceV1 []byte

func monthlyNFTTemplateSlotsV1(raw []byte) ([]MonthlyNFTSlotV1, error) {
	var source struct {
		Version        string `json:"version"`
		Year           int    `json:"year"`
		TotalPlanned   int    `json:"totalPlanned"`
		AwardsPerMonth int    `json:"awardsPerMonth"`
		Rarity         string `json:"rarity"`
		Items          []struct {
			StockNo      int    `json:"stockNo"`
			ID           string `json:"id"`
			Period       string `json:"period"`
			Name         string `json:"name"`
			Rarity       string `json:"rarity"`
			Concept      string `json:"concept"`
			ImagePath    string `json:"imagePath"`
			MetadataPath string `json:"metadataPath"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, fmt.Errorf("%w: stock JSON: %v", ErrMonthlyNFTInvalidPlan, err)
	}
	if source.Version != "NFT_MONTHLY_STOCK_PRE_MAINNET_V1" || source.Year != 2026 || source.TotalPlanned != 12 || source.AwardsPerMonth != 1 || source.Rarity != "rare" || len(source.Items) != 12 {
		return nil, fmt.Errorf("%w: stock must define twelve 2026 monthly slots", ErrMonthlyNFTInvalidPlan)
	}
	seenNames, seenConcepts := map[string]bool{}, map[string]bool{}
	slots := make([]MonthlyNFTSlotV1, 0, 12)
	for index, item := range source.Items {
		month := index + 1
		period := fmt.Sprintf("2026-%02d", month)
		id := "rare-" + period + "-001"
		image := "telegram_game/telegram_game/dev_ui/assets/nft/" + id + ".png"
		metadata := "nft/catalog/metadata/" + id + ".json"
		name, concept := strings.TrimSpace(item.Name), strings.TrimSpace(item.Concept)
		if item.StockNo != month || item.Period != period || item.ID != id || item.Rarity != "rare" || item.ImagePath != image || item.MetadataPath != metadata || name == "" || concept == "" || name != item.Name || concept != item.Concept || seenNames[name] || seenConcepts[concept] {
			return nil, fmt.Errorf("%w: invalid stock identity for month %d", ErrMonthlyNFTInvalidPlan, month)
		}
		seenNames[name], seenConcepts[concept] = true, true
		slots = append(slots, MonthlyNFTSlotV1{
			ID: id, Period: period, MonthIndex: month, Name: name,
			Rarity: "rare", Concept: concept, ImagePath: image, MetadataPath: metadata,
		})
	}
	return slots, nil
}
