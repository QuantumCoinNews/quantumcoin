package internal

import (
	"math"
)

// RewardConfig: Blok ödül/halving sistemi parametreleri
type RewardConfig struct {
	InitialReward int // İlk blok ödülü (QC)
	HalvingPeriod int // Kaç blokta bir halving
	TotalSupply   int // Maksimum toplam arz (opsiyonel)
}

// CalculateReward: Verilen blok yüksekliğinde ödülü döndürür
func CalculateReward(config RewardConfig, height int) int {
	halvings := height / config.HalvingPeriod
	reward := float64(config.InitialReward) / math.Pow(2, float64(halvings))
	if reward < 1 {
		return 0 // Artık ödül yok, zincir durdu (veya yalnızca fee ile devam)
	}
	return int(reward)
}

// Kullanım örneği:
// reward := internal.CalculateReward(internal.RewardConfig{InitialReward: 50, HalvingPeriod: 1051200}, blockHeight)
