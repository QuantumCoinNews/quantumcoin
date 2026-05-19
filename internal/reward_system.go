// internal/reward_system.go
package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

const (
	// RAM'de tutulacak maksimum bonus satırı
	maxMemBonusLines = 1000

	// Bonus log dosyasının maks. boyutu (10 MB). Aşıldığında .1 uzantılı rotate edilir.
	maxBonusFileSize = 10 * 1024 * 1024
)

var (
	bonusFilePath string

	memBonusMu  sync.Mutex
	memBonusLog []Bonus

	// Dosya yazımı için ayrı kilit (eşzamanlı append çakışmasın)
	bonusFileMu sync.Mutex
)

// SetBonusFile: Bonusların kaydedileceği dosya yolunu ayarlar.
// Güvenlik:
// - Boş/whitespace ise dosya log'u devre dışı.
// - Göreli path ise QC_NODE_DIR altında çözülür (release klasörü kök kabul edilir).
func SetBonusFile(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		bonusFilePath = ""
		return
	}

	// Göreli yol → QC_NODE_DIR altında çöz
	if !filepath.IsAbs(path) {
		if base := strings.TrimSpace(os.Getenv("QC_NODE_DIR")); base != "" {
			path = filepath.Join(base, path)
		}
	}

	bonusFilePath = path
}

// GiveBonus: bonus ödül dağıtır (RAM + dosya + core log).
// Not: Address boş veya amount <= 0 ise sessizce yok sayar (spam/bug engeli).
func GiveBonus(address, bonusType string, amount int, reason, metadata string) {
	address = strings.TrimSpace(address)
	bonusType = strings.TrimSpace(bonusType)

	if address == "" || amount <= 0 {
		return
	}

	b := Bonus{
		Address:   address,
		Type:      bonusType,
		Amount:    amount,
		Reason:    reason,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	// Konsola yaz (hızlı takip)
	fmt.Printf("[BONUS] %s → %d QC (%s)\n", b.Address, b.Amount, b.Type)
	if b.Reason != "" {
		fmt.Println(" Reason:", b.Reason)
	}

	// Çekirdek string-log (mevcut bonus_core yapını kullanıyoruz)
	// TxID bilgisi burada yoksa boş string geçiyoruz.
	GiveBonusCore(b.Address, b.Type, b.Amount, b.Reason, "")

	// Hafızaya ekle (son maxMemBonusLines satır tutulur)
	memBonusMu.Lock()
	memBonusLog = append(memBonusLog, b)
	if len(memBonusLog) > maxMemBonusLines {
		memBonusLog = memBonusLog[len(memBonusLog)-maxMemBonusLines:]
	}
	memBonusMu.Unlock()

	// Dosyaya kaydet (varsa)
	if bonusFilePath != "" {
		_ = SaveBonus(b)
	}
}

// ListBonuses: belirli adresin bonuslarını RAM'deki logtan döndürür.
// address boş ise tüm RAM logu döner.
func ListBonuses(address string) []Bonus {
	address = strings.TrimSpace(address)

	memBonusMu.Lock()
	defer memBonusMu.Unlock()

	var results []Bonus
	for _, b := range memBonusLog {
		if address == "" || b.Address == address {
			results = append(results, b)
		}
	}
	return results
}

// SaveBonus: bonusu JSON satırı olarak dosyaya ekler.
// Güvenlik:
// - Dosya 0600 (sadece kullanıcı).
// - Boyut > maxBonusFileSize ise eski dosya .1 uzantısıyla rotate edilir.
func SaveBonus(b Bonus) error {
	if bonusFilePath == "" {
		return nil
	}

	bonusFileMu.Lock()
	defer bonusFileMu.Unlock()

	// Basit rotate (hata vermez, sadece deneme)
	if info, err := os.Stat(bonusFilePath); err == nil && info.Size() > maxBonusFileSize {
		_ = os.Rename(bonusFilePath, bonusFilePath+".1")
	}

	f, err := os.OpenFile(bonusFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(b)
	if err != nil {
		return err
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}

	return nil
}
