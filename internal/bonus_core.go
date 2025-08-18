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

// GiveBonusCore — çekirdek bonus verme işlemi (AI veya blockchain bağımsız)
func GiveBonusCore(address, bonusType string, amount int, reason string, txID string) {
	bonusLogLock.Lock()
	defer bonusLogLock.Unlock()

	entry := fmt.Sprintf("[%s] Bonus to %s: %d QC | Type: %s | Reason: %s | TxID: %s",
		time.Now().Format(time.RFC3339), address, amount, bonusType, reason, txID)
	bonusLog = append(bonusLog, entry)

	fmt.Println("🎁 Bonus awarded:", entry)
}

// ListBonusLog — mevcut bonus loglarını döndürür
func ListBonusLog() []string {
	bonusLogLock.Lock()
	defer bonusLogLock.Unlock()

	// kopya döndür, dışarıdan manipüle edilmesin
	copyLog := make([]string, len(bonusLog))
	copy(copyLog, bonusLog)
	return copyLog
}
