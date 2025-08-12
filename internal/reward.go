package internal

import (
	"math"
	"time"
)

// RewardConfig: Blok bazlı halving (height) veya zaman bazlı halving (timestamp) için ayarlar.
// İkisi de doluysa, zaman bazlı olan önceliklidir.
type RewardConfig struct {
	InitialReward       int   // İlk blok ödülü (QC)
	HalvingPeriodBlocks int   // Kaç blokta bir halving (blok bazlı)
	HalvingPeriodSecs   int64 // Kaç saniyede bir halving (zaman bazlı)
	MiningPeriodSecs    int64 // Madencilik süresi (zaman bazlı; 0 = sınırsız)
	GenesisUnix         int64 // Genesis unix (zaman bazlı)

	TotalSupply int // Maks toplam arz (0 = sınırsız)
}

// CalculateRewardByHeight: Blok yüksekliğine göre halving
func CalculateRewardByHeight(initialReward int, halvingPeriodBlocks int, height int) int {
	if initialReward <= 0 || halvingPeriodBlocks <= 0 || height < 0 {
		return 0
	}
	halvings := height / halvingPeriodBlocks
	reward := float64(initialReward) / math.Pow(2, float64(halvings))
	if reward < 1 {
		return 0
	}
	return int(reward)
}

// CalculateRewardByTimeNow: Zaman bazlı halving (şimdiye göre)
func CalculateRewardByTimeNow(initialReward int, genesisUnix, halvingPeriodSecs, miningPeriodSecs int64) int {
	return CalculateRewardByTimeAt(initialReward, genesisUnix, halvingPeriodSecs, miningPeriodSecs, time.Now().Unix())
}

// CalculateRewardByTimeAt: Zaman bazlı halving (belirli bir zaman için)
func CalculateRewardByTimeAt(initialReward int, genesisUnix, halvingPeriodSecs, miningPeriodSecs, now int64) int {
	if initialReward <= 0 || halvingPeriodSecs <= 0 {
		return 0
	}
	elapsed := now - genesisUnix
	if elapsed < 0 {
		elapsed = 0
	}
	if miningPeriodSecs > 0 && elapsed > miningPeriodSecs {
		return 0
	}
	halvings := int(elapsed / halvingPeriodSecs)
	reward := float64(initialReward) / math.Pow(2, float64(halvings))
	if reward < 1 {
		return 0
	}
	return int(reward)
}

// ClampToSupply: Önerilen ödülü arz tavanının üzerinde değilse döndürür,
// aşıyorsa kalan arz kadar kısıtlar (0 → sınırsız).
func ClampToSupply(suggested, totalMinted, totalSupply int) int {
	if suggested <= 0 {
		return 0
	}
	if totalSupply <= 0 {
		return suggested // sınırsız
	}
	remaining := totalSupply - totalMinted
	if remaining <= 0 {
		return 0
	}
	if suggested > remaining {
		return remaining
	}
	return suggested
}

// ComputeReward: RewardConfig'e göre (zaman/height) hesapla, sonra arz tavanına göre kırp.
// height: kazılacak bloğun yüksekliği (genelde tip+1), now: unix time (0 ise time.Now()).
func ComputeReward(cfg RewardConfig, height int, totalMinted int, now int64) int {
	var base int
	if cfg.HalvingPeriodSecs > 0 && cfg.GenesisUnix > 0 {
		if now == 0 {
			now = time.Now().Unix()
		}
		base = CalculateRewardByTimeAt(cfg.InitialReward, cfg.GenesisUnix, cfg.HalvingPeriodSecs, cfg.MiningPeriodSecs, now)
	} else if cfg.HalvingPeriodBlocks > 0 {
		base = CalculateRewardByHeight(cfg.InitialReward, cfg.HalvingPeriodBlocks, height)
	} else {
		// Halving devre dışı: sabit ödül
		base = cfg.InitialReward
	}
	return ClampToSupply(base, totalMinted, cfg.TotalSupply)
}
