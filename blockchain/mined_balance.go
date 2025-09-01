package blockchain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type minedBal struct {
	Address string  `json:"address"`
	Balance float64 `json:"balance"`
	Updated int64   `json:"updated"`
}

func exeDir() string {
	self, _ := os.Executable()
	return filepath.Dir(self)
}
func minedBalancePath() string { return filepath.Join(exeDir(), "mined_balance.json") }

// Her başarılı blokta ödülü release\mined_balance.json'a kümülatif yazar.
func AddMinedBalance(addr string, deltaQC int) {
	addr = strings.TrimSpace(addr)
	if addr == "" || deltaQC == 0 {
		return
	}

	p := minedBalancePath()
	var m minedBal
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &m)
	}

	// Tek adres takip: adres değişirse sayaç sıfırlanır
	if !strings.EqualFold(strings.TrimSpace(m.Address), addr) {
		m.Address = addr
		m.Balance = 0
	}

	m.Balance += float64(deltaQC)
	m.Updated = time.Now().Unix()

	tmp := p + ".tmp"
	if b, err := json.MarshalIndent(m, "", "  "); err == nil {
		_ = os.WriteFile(tmp, b, 0644)
		_ = os.Rename(tmp, p) // atomik güncelleme
	}
}
