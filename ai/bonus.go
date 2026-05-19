package ai

import (
	"sync"
	"time"
)

// Zincir / madenci için AI bonus durumunun özeti.
type AIBonus struct {
	TotalBonusQC float64 `json:"total_bonus_qc"` // bugüne kadar toplam
	LastBonusQC  float64 `json:"last_bonus_qc"`  // son bonus miktarı
	Reason       string  `json:"reason"`         // son bonus sebebi
	UpdatedAt    int64   `json:"updated_at"`     // Unix time
}

var (
	bonusMu    sync.RWMutex
	bonusState AIBonus
)

// Bonus durumunu dışarıdan tamamen set etmek istersen:
func SetBonus(totalBonus, lastBonus float64, reason string) {
	if !Enabled() {
		return
	}

	bonusMu.Lock()
	defer bonusMu.Unlock()

	bonusState = AIBonus{
		TotalBonusQC: totalBonus,
		LastBonusQC:  lastBonus,
		Reason:       reason,
		UpdatedAt:    time.Now().Unix(),
	}
}

// Tipik kullanım: "AI yeni bir bonus hesapladı; toplamı arttır"
func AddBonus(delta float64, reason string) {
	if !Enabled() {
		return
	}

	bonusMu.Lock()
	defer bonusMu.Unlock()

	bonusState.TotalBonusQC += delta
	bonusState.LastBonusQC = delta
	bonusState.Reason = reason
	bonusState.UpdatedAt = time.Now().Unix()
}

// HTTP endpoint'in kullanacağı snapshot.
func GetBonusSnapshot() AIBonus {
	bonusMu.RLock()
	defer bonusMu.RUnlock()
	return bonusState
}

// persist yükleme için
func setBonusFromSnapshot(b AIBonus) {
	bonusMu.Lock()
	defer bonusMu.Unlock()
	bonusState = b
}
