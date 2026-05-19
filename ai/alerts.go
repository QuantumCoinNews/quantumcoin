package ai

import (
	"sync"
	"time"
)

// Uyarı seviyesi
type AlertLevel string

const (
	AlertInfo    AlertLevel = "info"
	AlertWarning AlertLevel = "warning"
	AlertError   AlertLevel = "error"
)

// AI'nın tespit ettiği tekil uyarı kaydı.
type AIAlert struct {
	ID        string     `json:"id"`
	Level     AlertLevel `json:"level"`
	Code      string     `json:"code"`
	Message   string     `json:"message"`
	Source    string     `json:"source"`     // "miner", "mempool", "p2p" vb.
	CreatedAt int64      `json:"created_at"` // Unix time
}

var (
	alertsMu     sync.RWMutex
	alertsBuffer []AIAlert
	maxAlerts    = 100
)

// Node içinden yeni bir uyarı ekleneceği zaman çağırılır.
// Örn: ai.PushAlert(ai.AlertWarning, "HIGH_FEE", "Suspiciously high fee", "mempool")
func PushAlert(level AlertLevel, code, message, source string) {
	// AI tamamen kapalıysa hiçbir şey yapma
	if !Enabled() {
		return
	}

	a := AIAlert{
		ID:        newAlertID(),
		Level:     level,
		Code:      code,
		Message:   message,
		Source:    source,
		CreatedAt: time.Now().Unix(),
	}

	alertsMu.Lock()
	defer alertsMu.Unlock()

	alertsBuffer = append(alertsBuffer, a)
	if len(alertsBuffer) > maxAlerts {
		alertsBuffer = alertsBuffer[len(alertsBuffer)-maxAlerts:]
	}
}

// Tüm uyarıların snapshot'ı (kopya)
func GetAlertsSnapshot() []AIAlert {
	alertsMu.RLock()
	defer alertsMu.RUnlock()

	out := make([]AIAlert, len(alertsBuffer))
	copy(out, alertsBuffer)
	return out
}

// State yükleme (persist için kullanacağız)
func setAlertsFromSnapshot(alerts []AIAlert) {
	alertsMu.Lock()
	defer alertsMu.Unlock()

	alertsBuffer = make([]AIAlert, len(alerts))
	copy(alertsBuffer, alerts)
	if len(alertsBuffer) > maxAlerts {
		alertsBuffer = alertsBuffer[len(alertsBuffer)-maxAlerts:]
	}
}

// Basit ID üretici
func newAlertID() string {
	return time.Now().Format("20060102T150405")
}
