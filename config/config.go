package config

import "math/big"

const (
	// İlk blok ödülü (QC)
	InitialBlockReward = 50

	// Halving aralığı (örnek: 2 yıl = 525600 dakika)
	HalvingInterval = 525600

	// Blok başına maksimum işlem
	MaxTransactionsPerBlock = 1000

	// Genesis bloku coinbase datası
	GenesisCoinbaseData = "QuantumCoin Genesis Block"

	// PoW zorluk seviyesi (2^256 / 2^TargetBits)
	TargetBits = 20

	// Maksimum toplam arz (QC)
	MaxSupply = 25_500_000

	// Ana ağ adı/versiyonu
	NetworkID = "QuantumNet-1"

	// DevWallet = "qc1..." // Geliştirici cüzdan adresi

	// İleri parametreler için örnek:
	// NFTMetadataBaseURL = "https://quantumcoin.io/nft/"
	// StakeMinDuration   = 60 * 60 * 24 * 7 // 1 hafta saniye
	// SupportedLanguages = "en,tr,es,zh"
)

// Target: PoW için zorluk hedefi
var Target *big.Int

func init() {
	Target = big.NewInt(1)
	Target.Lsh(Target, 256-TargetBits) // Target = 1 << (256 - TargetBits)
}
