// internal/bonus_core.go
package internal

import (
	"fmt"
	"sync"
	"time"
)

var (
	bonusLog     []string
	bonusLogLock sync.Mutex
)

// GiveBonusCore — çekirdek bonus verme işlemi (AI veya blockchain bağımsız, string log).
func GiveBonusCore(address, bonusType string, amount int, reason string, txID string) {
	bonusLogLock.Lock()
	defer bonusLogLock.Unlock()

	entry := fmt.Sprintf(
		"[%s] Bonus to %s: %d QC | Type: %s | Reason: %s | TxID: %s",
		time.Now().Format(time.RFC3339),
		address,
		amount,
		bonusType,
		reason,
		txID,
	)
	bonusLog = append(bonusLog, entry)

	fmt.Println("🎁 Bonus awarded:", entry)
}

// ListBonusLog — mevcut bonus loglarını döndürür (kopya).
func ListBonusLog() []string {
	bonusLogLock.Lock()
	defer bonusLogLock.Unlock()

	copyLog := make([]string, len(bonusLog))
	copy(copyLog, bonusLog)
	return copyLog
}
