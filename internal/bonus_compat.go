// internal/reward_system.go
package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Bonus struct {
	Address   string    `json:"address"`
	Type      string    `json:"type"`
	Amount    int       `json:"amount"`
	Reason    string    `json:"reason,omitempty"`
	Metadata  string    `json:"metadata,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	bonusFilePath string

	memBonusMu  sync.Mutex
	memBonusLog []Bonus
)

// SetBonusFile: Bonusların kaydedileceği dosya yolunu ayarlar
func SetBonusFile(path string) {
	bonusFilePath = path
}

// GiveBonus: bonus ödül dağıtır
func GiveBonus(address, bonusType string, amount int, reason, metadata string) {
	b := Bonus{
		Address:   address,
		Type:      bonusType,
		Amount:    amount,
		Reason:    reason,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	// Konsola yaz
	fmt.Printf("[BONUS] %s → %d QC (%s)\n", address, amount, bonusType)
	if reason != "" {
		fmt.Println(" Reason:", reason)
	}

	// Hafızaya ekle
	memBonusMu.Lock()
	memBonusLog = append(memBonusLog, b)
	memBonusMu.Unlock()

	// Dosyaya kaydet
	if bonusFilePath != "" {
		_ = SaveBonus(b)
	}
}

// ListBonuses: belirli adresin bonuslarını döndürür
func ListBonuses(address string) []Bonus {
	memBonusMu.Lock()
	defer memBonusMu.Unlock()

	var results []Bonus
	for _, b := range memBonusLog {
		if b.Address == address {
			results = append(results, b)
		}
	}
	return results
}

// SaveBonus: bonusu JSON olarak dosyaya ekler
func SaveBonus(b Bonus) error {
	if bonusFilePath == "" {
		return nil
	}

	f, err := os.OpenFile(bonusFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, _ := json.Marshal(b)
	_, err = f.Write(append(data, '\n'))
	return err
}
