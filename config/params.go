package config

// Dinamik sistem parametreleri (runtime'da değişebilir)
var (
	BlockTimeSeconds        = 30        // Blok üretim hedef süresi (saniye)
	MaxBlockSizeBytes       = 1_000_000 // 1 MB blok limiti
	EnableStaking           = true      // Stake açık mı?
	EnableMining            = true      // Madencilik açık mı?
	CurrentMiningDifficulty = 16        // Anlık PoW zorluk seviyesi
	CurrentStakeRewardRate  = 0.05      // Yıllık stake ödül oranı (%5)
)
