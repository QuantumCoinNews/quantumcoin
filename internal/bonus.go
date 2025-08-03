package internal

import (
	"fmt"
	"quantumcoin/ai"
	"quantumcoin/blockchain"
	"sync"
	"time"
)

// Bonus tipleri
const (
	BonusTypeAI    = "AI"
	BonusTypeEvent = "Event"
)

// Bonus kayıt yapısı
type BonusRecord struct {
	Address     string    `json:"address"`
	Type        string    `json:"type"`
	Amount      int       `json:"amount"`
	Description string    `json:"description"`
	Metadata    string    `json:"metadata"`
	Timestamp   time.Time `json:"timestamp"`
}

// Bonusları tutan global slice
var (
	bonusLog []BonusRecord
	mu       sync.Mutex
)

// Bonus ekleyen fonksiyon
func GiveBonus(address string, bonusType string, amount int, description string, metadata string) {
	mu.Lock()
	defer mu.Unlock()
	rec := BonusRecord{
		Address:     address,
		Type:        bonusType,
		Amount:      amount,
		Description: description,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}
	bonusLog = append(bonusLog, rec)
	fmt.Printf("BONUS: %s adresine %d adet %s (%s)\n", address, amount, bonusType, description)
}

// Adrese göre (veya hepsini) dönen fonksiyon
func ListBonuses(address string) []BonusRecord {
	mu.Lock()
	defer mu.Unlock()
	var filtered []BonusRecord
	for _, br := range bonusLog {
		if address == "" || br.Address == address {
			filtered = append(filtered, br)
		}
	}
	return filtered
}

// Otomatik AI bonus dağıtımı (mevcut kodun)
func DistributeAIBonuses(txs []*blockchain.Transaction) {
	// Anomaliye göre bonus (güvenlik ödülü)
	anomalies := ai.AnalyzeTransactions(txs, 5, 24)
	for _, report := range anomalies {
		if report.Suspicious {
			GiveBonus(
				report.WalletAddress,
				BonusTypeAI,
				2,
				"AI şüpheli işlem sonrası bilinçlendirme bonusu",
				"",
			)
			fmt.Printf("Anomaly bonus: %v\n", report)
		}
	}

	// Davranışsal öneriye göre bonus
	recs := ai.GenerateRecommendations(txs, 14, 10)
	for _, rec := range recs {
		if rec.Score > 0.7 {
			GiveBonus(
				rec.WalletAddress,
				BonusTypeAI,
				1,
				"AI analizine göre aktiflik/ödül bonusu",
				"",
			)
			fmt.Printf("Recommendation bonus: %v\n", rec)
		}
	}

	// Optimize ödül
	suggestions := ai.OptimizeRewards(txs, 10, 1)
	for _, sug := range suggestions {
		amount := int(sug.SuggestedReward)
		if amount > 0 {
			GiveBonus(
				sug.WalletAddress,
				BonusTypeAI,
				amount,
				sug.Reason,
				"",
			)
			fmt.Printf("Optimized reward bonus: %v\n", sug)
		}
	}
}
