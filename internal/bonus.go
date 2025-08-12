package internal

import (
	"encoding/json"
	"fmt"
	"os"
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

// varsayılan dosya; SetBonusFile ile override edilebilir
var (
	bonusFilePath = "bonus_store.json"
	bonusLog      []BonusRecord
	mu            sync.Mutex
)

// SetBonusFile: bonus kayıt dosyasını çalışma zamanında değiştir
// Not: init'te okunan önceki kayıtlar bellekte kalır, bundan sonrası yeni dosyaya yazılır.
func SetBonusFile(path string) {
	if path == "" {
		return
	}
	mu.Lock()
	bonusFilePath = path
	mu.Unlock()
}

func init() { // açılışta varsa dosyadan yükle
	_ = loadBonusLog()
}

// Bonus kayıt yapısı
type BonusRecord struct {
	Address     string    `json:"address"`
	Type        string    `json:"type"`
	Amount      int       `json:"amount"`
	Description string    `json:"description"`
	Metadata    string    `json:"metadata"`
	Timestamp   time.Time `json:"timestamp"`
}

// --- Kalıcılık yardımcıları ---

func loadBonusLog() error {
	mu.Lock()
	defer mu.Unlock()
	bonusLog = nil
	data, err := os.ReadFile(bonusFilePath)
	if err != nil {
		return nil // dosya yoksa sorun değil
	}
	return json.Unmarshal(data, &bonusLog)
}

func persistBonusLog() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(bonusLog, "", "  ")
	_ = os.WriteFile(bonusFilePath, data, 0644)
}

// --- Public API ---

// GiveBonus: bonus kaydı ekler ve diske yazar
func GiveBonus(address, bonusType string, amount int, description, metadata string) {
	mu.Lock()
	rec := BonusRecord{
		Address:     address,
		Type:        bonusType,
		Amount:      amount,
		Description: description,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}
	bonusLog = append(bonusLog, rec)
	mu.Unlock()

	fmt.Printf("BONUS: %s adresine %d adet %s (%s)\n", address, amount, bonusType, description)
	persistBonusLog()
}

// ListBonuses: adrese göre (veya hepsi) bonusları döner
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

// DistributeAIBonuses: AI analizlerinden bonus dağıt
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
