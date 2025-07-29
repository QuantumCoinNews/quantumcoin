package config

const (
	// Sürüm & build bilgisi
	Version = "QuantumCoin v2.0.0"
	Build   = "2025-07-Quantum"

	// i18n: Desteklenen diller
	LangEN = "en" // English
	LangTR = "tr" // Türkçe
	LangES = "es" // Español
	LangZH = "zh" // 中文

	// Proje sloganı (i18n ile değiştirilebilir)
	Slogan = "Yapılmayanı yapmak." // Çoklu dil için SloganEN, SloganTR vs. düşünebilirsin

	// Explorer & UI temaları
	ThemeDark  = "dark"
	ThemeLight = "light"

	// NFT metadata JSON API base URL
	NFTBaseURL = "https://nft.quantumcoin.org/metadata/"

	// Stake süresi sabitleri (gün cinsinden)
	StakeShortTerm = 90  // 3 ay
	StakeMidTerm   = 180 // 6 ay
	StakeLongTerm  = 365 // 1 yıl

	// Yeni sabitler ekleyebilirsin:
	// BurnAddress = "qc1burn000..." // Yakım adresi
	// TokenFee    = 100 // QC cinsinden token çıkarma ücreti
)
