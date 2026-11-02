// internal/reward_system.go

// Package internal contains QuantumCoin project code for this module.
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
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

const (
	maxMemBonusLines = 1000
	maxBonusFileSize = 10 * 1024 * 1024
)

var (
	bonusPathMu   sync.RWMutex
	bonusFilePath string
	memBonusMu    sync.Mutex
	memBonusLog   []Bonus
	bonusFileMu   sync.Mutex
)

// SetBonusFile bonus kayıtlarının yazılacağı dosya yolunu ayarlar.
func SetBonusFile(path string) {
	path = strings.TrimSpace(path)

	if path != "" && !filepath.IsAbs(path) {
		base := strings.TrimSpace(os.Getenv("QC_NODE_DIR"))
		if base == "" {
			exe, err := os.Executable()
			if err == nil {
				base = filepath.Dir(exe)
			}
		}
		if base == "" {
			base, _ = os.Getwd()
		}
		path = filepath.Join(base, path)
	}

	bonusPathMu.Lock()
	bonusFilePath = path
	bonusPathMu.Unlock()
}

func currentBonusFilePath() string {
	bonusPathMu.RLock()
	defer bonusPathMu.RUnlock()
	return bonusFilePath
}

// GiveBonus settlement öncesi bonus kaydı oluşturur.
// Blockchain transferi veya mint işlemi yapmaz.
func GiveBonus(
	address,
	bonusType string,
	amount int,
	reason,
	metadata string,
) bool {
	address = strings.TrimSpace(address)
	bonusType = strings.TrimSpace(bonusType)

	if address == "" || amount <= 0 {
		return false
	}

	b := Bonus{
		Address:   address,
		Type:      bonusType,
		Amount:    amount,
		Reason:    reason,
		Metadata:  metadata,
		Status:    "pending_settlement",
		Timestamp: time.Now(),
	}

	path := currentBonusFilePath()
	if path != "" {
		if err := SaveBonus(b); err != nil {
			fmt.Printf(
				"[BONUS] kayıt başarısız; settlement beklemede: %v\n",
				err,
			)
			return false
		}
	}

	GiveBonusCore(b.Address, b.Type, b.Amount, b.Reason, "")

	memBonusMu.Lock()
	memBonusLog = append(memBonusLog, b)
	if len(memBonusLog) > maxMemBonusLines {
		memBonusLog = memBonusLog[len(memBonusLog)-maxMemBonusLines:]
	}
	memBonusMu.Unlock()

	fmt.Printf(
		"[BONUS] recorded; settlement pending: %s → %d QC (%s)\n",
		b.Address,
		b.Amount,
		b.Type,
	)

	if b.Reason != "" {
		fmt.Println(" Reason:", b.Reason)
	}

	return true
}

// ListBonuses belirli adrese ait bonus kayıtlarını döndürür.
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

// SaveBonus bonusu JSON satırı olarak dosyaya ekler.
func SaveBonus(b Bonus) error {
	path := currentBonusFilePath()
	if path == "" {
		return nil
	}

	bonusFileMu.Lock()
	defer bonusFileMu.Unlock()

	parent := filepath.Dir(path)
	if parent != "." && parent != "" {
		if err := os.MkdirAll(parent, 0o700); err != nil {
			return fmt.Errorf("bonus parent directory: %w", err)
		}
	}

	if info, err := os.Stat(path); err == nil &&
		info.Size() > maxBonusFileSize {
		if err := os.Rename(path, path+".1"); err != nil {
			return fmt.Errorf("bonus log rotate: %w", err)
		}
	}

	f, err := os.OpenFile(
		path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("bonus log open: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("bonus encode: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("bonus log append: %w", err)
	}

	return nil
}
