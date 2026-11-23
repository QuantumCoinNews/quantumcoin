package ai

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// AlertLevel documents the QuantumCoin component used by this module.
// Uyarı seviyesi
type AlertLevel string

const (
	AlertInfo    AlertLevel = "info"
	AlertWarning AlertLevel = "warning"
	AlertError   AlertLevel = "error"
)

// AIAlert documents the QuantumCoin component used by this module.
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
	alertsMu      sync.RWMutex
	alertsBuffer  []AIAlert
	maxAlerts     = 100
	alertSequence atomic.Uint64
	alertPrefix   = func() string {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err == nil {
			return hex.EncodeToString(nonce[:])
		}
		return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	}()
)

// PushAlert documents the QuantumCoin component used by this module.
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
		Code:      boundedText(code),
		Message:   boundedText(message),
		Source:    boundedText(source),
		CreatedAt: time.Now().Unix(),
	}

	alertsMu.Lock()
	defer alertsMu.Unlock()

	alertsBuffer = append(alertsBuffer, a)
	if len(alertsBuffer) > maxAlerts {
		alertsBuffer = alertsBuffer[len(alertsBuffer)-maxAlerts:]
	}
}

// GetAlertsSnapshot documents the QuantumCoin component used by this module.
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

	if len(alerts) > maxAlerts {
		alerts = alerts[len(alerts)-maxAlerts:]
	}
	alertsBuffer = make([]AIAlert, len(alerts))
	copy(alertsBuffer, alerts)
	seen := make(map[string]bool)
	for i := range alertsBuffer {
		a := &alertsBuffer[i]
		if a.ID == "" || seen[a.ID] {
			a.ID = newAlertID()
		}
		seen[a.ID] = true
		a.Code, a.Message, a.Source = boundedText(a.Code), boundedText(a.Message), boundedText(a.Source)
	}
	if len(alertsBuffer) > maxAlerts {
		alertsBuffer = alertsBuffer[len(alertsBuffer)-maxAlerts:]
	}
}

// Basit ID üretici
func newAlertID() string {
	return fmt.Sprintf("%s-%d", alertPrefix, alertSequence.Add(1))
}
