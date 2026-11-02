package nft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMonthlyCatalogPlanMatchesAllViewsV1(t *testing.T) {
	stockRaw, err := os.ReadFile("catalog/monthly_rare_nft_stock_2026.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stockRaw, monthlyNFTStock2026SourceV1) {
		t.Fatal("compiled stock differs from checked-in stock")
	}
	var stock struct {
		Items []struct {
			StockNo      int    `json:"stockNo"`
			ID           string `json:"id"`
			Period       string `json:"period"`
			Name         string `json:"name"`
			Concept      string `json:"concept"`
			Rarity       string `json:"rarity"`
			ImagePath    string `json:"imagePath"`
			MetadataPath string `json:"metadataPath"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stockRaw, &stock); err != nil {
		t.Fatal(err)
	}
	plan := BuildMonthlyRareNFTPlaceholderPlanV1(2026)
	if err := ValidateMonthlyNFTPlanV1(plan); err != nil {
		t.Fatal(err)
	}
	if len(stock.Items) != len(plan.Slots) {
		t.Fatal("stock/Go slot count drift")
	}
	for i, item := range stock.Items {
		expected := MonthlyNFTSlotV1{ID: item.ID, Period: item.Period, MonthIndex: item.StockNo, Name: item.Name,
			Concept: item.Concept, Rarity: item.Rarity, ImagePath: item.ImagePath, MetadataPath: item.MetadataPath}
		if plan.Slots[i] != expected {
			t.Fatalf("month %d Go/stock drift: got %+v, want %+v", i+1, plan.Slots[i], expected)
		}
	}
	raw, err := os.ReadFile("catalog/monthly_rare_nft_2026_placeholder_plan.json")
	if err != nil {
		t.Fatal(err)
	}
	var checkedIn MonthlyNFTPlanV1
	if err := json.Unmarshal(raw, &checkedIn); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan, checkedIn) {
		t.Fatal("Go plan and checked-in JSON plan differ")
	}
	if plan.Slots[5].Name != "Quantum Lunar Pioneer" {
		t.Fatal("existing June identity moved")
	}
}

func TestMonthlyCatalogAllMetadataFilesMatchStockV1(t *testing.T) {
	plan := BuildMonthlyRareNFTPlaceholderPlanV1(2026)
	files, err := filepath.Glob("catalog/metadata/rare-2026-*-001.json")
	if err != nil || len(files) != 12 {
		t.Fatalf("expected twelve metadata files, got %d: %v", len(files), err)
	}
	for _, slot := range plan.Slots {
		raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(slot.MetadataPath)))
		if err != nil {
			t.Fatalf("%s: metadata missing: %v", slot.Period, err)
		}
		var metadata struct {
			Name        string `json:"name"`
			Period      string `json:"period"`
			Rarity      string `json:"rarity"`
			Description string `json:"description"`
			Attributes  []struct {
				Type  string `json:"trait_type"`
				Value string `json:"value"`
			} `json:"attributes"`
			SafeBoundary map[string]bool `json:"safeBoundary"`
		}
		if err := json.Unmarshal(raw, &metadata); err != nil {
			t.Fatal(err)
		}
		if metadata.Name != slot.Name || metadata.Period != slot.Period || metadata.Rarity != slot.Rarity || metadata.Description == "" {
			t.Fatalf("%s: metadata identity drift", slot.Period)
		}
		if len(metadata.SafeBoundary) != 4 {
			t.Fatal("metadata safe boundary fields missing")
		}
		for _, key := range []string{"minted", "chainMetadataWritten", "balanceChanged", "qcSupplyChanged"} {
			value, found := metadata.SafeBoundary[key]
			if !found || value {
				t.Fatalf("%s: unsafe metadata boundary %s", slot.Period, key)
			}
		}
		notMinted := false
		for _, attribute := range metadata.Attributes {
			if attribute.Type == "Mint Status" && attribute.Value == "Not Minted" {
				notMinted = true
			}
		}
		if !notMinted {
			t.Fatal("draft metadata must explicitly say Not Minted")
		}
	}
}

func TestMonthlyCatalogPreviewIsConsistentSubsetV1(t *testing.T) {
	raw, err := os.ReadFile("catalog/rare_nft_catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var preview struct {
		Items []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Period   string `json:"period"`
			Rarity   string `json:"rarity"`
			Image    string `json:"image"`
			Metadata string `json:"metadata"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &preview); err != nil {
		t.Fatal(err)
	}
	byID := map[string]MonthlyNFTSlotV1{}
	for _, slot := range BuildMonthlyRareNFTPlaceholderPlanV1(2026).Slots {
		byID[slot.ID] = slot
	}
	seen := map[string]bool{}
	for _, item := range preview.Items {
		slot, found := byID[item.ID]
		if !found || seen[item.ID] || item.Name != slot.Name || item.Period != slot.Period || item.Rarity != slot.Rarity || item.Image != slot.ImagePath || item.Metadata != slot.MetadataPath {
			t.Fatal("preview catalogue drift")
		}
		seen[item.ID] = true
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(item.Metadata))); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMonthlyCatalogTemplateRejectsMalformedIdentitiesV1(t *testing.T) {
	mutations := []struct {
		name   string
		change func(map[string]interface{})
	}{
		{"thirteen", func(s map[string]interface{}) {
			s["items"] = append(s["items"].([]interface{}), s["items"].([]interface{})[0])
		}},
		{"two_awards", func(s map[string]interface{}) { s["awardsPerMonth"] = 2 }},
		{"wrong_period", func(s map[string]interface{}) {
			s["items"].([]interface{})[0].(map[string]interface{})["period"] = "2026-06"
		}},
		{"wrong_id", func(s map[string]interface{}) {
			s["items"].([]interface{})[0].(map[string]interface{})["id"] = "rare-2026-06-001"
		}},
		{"duplicate_name", func(s map[string]interface{}) {
			items := s["items"].([]interface{})
			items[1].(map[string]interface{})["name"] = items[0].(map[string]interface{})["name"]
		}},
		{"duplicate_concept", func(s map[string]interface{}) {
			items := s["items"].([]interface{})
			items[1].(map[string]interface{})["concept"] = items[0].(map[string]interface{})["concept"]
		}},
		{"empty_concept", func(s map[string]interface{}) { s["items"].([]interface{})[0].(map[string]interface{})["concept"] = "" }},
		{"outside_metadata", func(s map[string]interface{}) {
			s["items"].([]interface{})[0].(map[string]interface{})["metadataPath"] = "../outside.json"
		}},
		{"wrong_image", func(s map[string]interface{}) {
			s["items"].([]interface{})[0].(map[string]interface{})["imagePath"] = "other.png"
		}},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			var source map[string]interface{}
			if err := json.Unmarshal(monthlyNFTStock2026SourceV1, &source); err != nil {
				t.Fatal(err)
			}
			test.change(source)
			raw, err := json.Marshal(source)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := monthlyNFTTemplateSlotsV1(raw); err == nil {
				t.Fatal("invalid template accepted")
			}
		})
	}
	if _, err := monthlyNFTTemplateSlotsV1([]byte("{")); err == nil {
		t.Fatal("malformed JSON accepted")
	}
}

func TestMonthlyCatalogTemplateYearProjectionStaysDisabledV1(t *testing.T) {
	for _, year := range []int{2026, 2027, 2032} {
		plan := BuildMonthlyRareNFTPlaceholderPlanV1(year)
		if err := ValidateMonthlyNFTPlanV1(plan); err != nil {
			t.Fatal(err)
		}
		for month, slot := range plan.Slots {
			period := fmt.Sprintf("%04d-%02d", year, month+1)
			if slot.Period != period || slot.ID != "rare-"+period+"-001" || slot.PreparedBeforeMainnet || slot.MintEnabled || slot.Minted {
				t.Fatal("invalid projected placeholder")
			}
		}
	}
}
