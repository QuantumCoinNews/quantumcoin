package miner

import "time"

// kaba bir hashrate tahmini (sadece görsel): süre ve zorluk bitlerinden türetilmiş
func estimateHashrate(elapsed time.Duration, difficultyBits int) float64 {
	if elapsed <= 0 {
		return 0
	}
	// 2^difficulty kadar arama varsayımı (çok kabaca, sadece görsel amaçlı)
	// NOT: Bu gerçek bir ölçüm değildir; PoW’nun “Run()” içindeki deneme sayısı bizde yok.
	work := 1 << uint(difficultyBits/2) // “yumuşatılmış” ölçek
	return float64(work) / elapsed.Seconds()
}
