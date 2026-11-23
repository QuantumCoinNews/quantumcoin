package ai

import (
	"math"
	"sync"
	"time"
)

// TelemetrySnapshot documents the QuantumCoin component used by this module.
// Ağ durumunun özet hali.
// /api/telemetry JSON output'u doğrudan buradan gelecek.
type TelemetrySnapshot struct {
	Timestamp         int64   `json:"timestamp"`  // Unix time (saniye)
	PeerCount         int     `json:"peer_count"` // aktif peer sayısı
	Hashrate          float64 `json:"hashrate"`   // toplam hashrate (hash/s ya da istediğin metrik)
	HashrateAvailable bool    `json:"hashrate_available"`
	// İstersen sonra buraya ekstra alanlar ekleyebiliriz (mempool_size, difficulty vs.)
}

var (
	telemetryMu   sync.RWMutex
	lastTelemetry TelemetrySnapshot
)

// UpdateTelemetry documents the QuantumCoin component used by this module.
// Node içinden (P2P, miner vs.) çağıracağın fonksiyon.
// Şimdilik sadece peerCount ve hashrate alıyor; ihtiyaç olursa signature'ı genişletiriz.
func UpdateTelemetry(peerCount int, hashrate float64) {
	if peerCount < 0 {
		peerCount = 0
	}
	available := !math.IsNaN(hashrate) && !math.IsInf(hashrate, 0) && hashrate >= 0
	if !available {
		hashrate = 0
	}
	telemetryMu.Lock()
	defer telemetryMu.Unlock()

	lastTelemetry = TelemetrySnapshot{
		Timestamp:         time.Now().Unix(),
		PeerCount:         peerCount,
		Hashrate:          hashrate,
		HashrateAvailable: available,
	}
}

func validTelemetry(t TelemetrySnapshot) bool {
	return t.PeerCount >= 0 && t.Timestamp >= 0 && !math.IsNaN(t.Hashrate) && !math.IsInf(t.Hashrate, 0) && t.Hashrate >= 0
}

// GetTelemetrySnapshot documents the QuantumCoin component used by this module.
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
