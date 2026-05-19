package ai

import (
	"sync"
	"time"
)

// Ağ durumunun özet hali.
// /api/telemetry JSON output'u doğrudan buradan gelecek.
type TelemetrySnapshot struct {
	Timestamp int64   `json:"timestamp"`  // Unix time (saniye)
	PeerCount int     `json:"peer_count"` // aktif peer sayısı
	Hashrate  float64 `json:"hashrate"`   // toplam hashrate (hash/s ya da istediğin metrik)
	// İstersen sonra buraya ekstra alanlar ekleyebiliriz (mempool_size, difficulty vs.)
}

var (
	telemetryMu   sync.RWMutex
	lastTelemetry TelemetrySnapshot
)

// Node içinden (P2P, miner vs.) çağıracağın fonksiyon.
// Şimdilik sadece peerCount ve hashrate alıyor; ihtiyaç olursa signature'ı genişletiriz.
func UpdateTelemetry(peerCount int, hashrate float64) {
	telemetryMu.Lock()
	defer telemetryMu.Unlock()

	lastTelemetry = TelemetrySnapshot{
		Timestamp: time.Now().Unix(),
		PeerCount: peerCount,
		Hashrate:  hashrate,
	}
}

// HTTP endpoint'in okuyacağı fonksiyon.
func GetTelemetrySnapshot() TelemetrySnapshot {
	telemetryMu.RLock()
	defer telemetryMu.RUnlock()
	return lastTelemetry
}

// persist yükleme için
func setTelemetryFromSnapshot(t TelemetrySnapshot) {
	telemetryMu.Lock()
	defer telemetryMu.Unlock()
	lastTelemetry = t
}
