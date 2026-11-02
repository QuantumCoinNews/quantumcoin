// internal/bonus_core.go

// Package internal contains QuantumCoin project code for this module.
package internal

import (
	"fmt"
	"sync"
)

var bonusLogLock sync.Mutex

// GiveBonusCore yalnızca konsola durum bilgisi yazar.
// Kanonik bonus RAM kaydı reward_system.go içindeki memBonusLog'dur.
// Blockchain transferi, mint veya bakiye değişikliği yapmaz.
func GiveBonusCore(
	address string,
	bonusType string,
	amount int,
	reason string,
	txID string,
) {
	status := "pending_settlement"
	if txID != "" {
		status = "settlement_referenced"
	}

	entry := fmt.Sprintf(
		"Bonus recorded: %d QC | Address: %s | Type: %s | Reason: %s | TxID: %s | Status: %s",
		amount,
		address,
		bonusType,
		reason,
		txID,
		status,
	)

	bonusLogLock.Lock()
	defer bonusLogLock.Unlock()

	fmt.Println("🎁 Bonus recorded; settlement pending:", entry)
}

// ListBonusLog geriye dönük uyumluluk için string görünümü döndürür.
// Veriler yalnızca reward_system.go içindeki sınırlı memBonusLog kaynağından üretilir.
func ListBonusLog() []string {
	memBonusMu.Lock()
	defer memBonusMu.Unlock()

	copyLog := make([]string, 0, len(memBonusLog))

	for _, bonus := range memBonusLog {
		copyLog = append(
			copyLog,
			fmt.Sprintf(
				"Bonus recorded: %d QC | Address: %s | Type: %s | Reason: %s | Status: %s",
				bonus.Amount,
				bonus.Address,
				bonus.Type,
				bonus.Reason,
				bonus.Status,
			),
		)
	}

	return copyLog
}
