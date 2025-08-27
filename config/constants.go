// config/constants.go
package config

const (
	// Sürüm & build bilgisi
	Version = "v0.1.0"     // testnet-alpha sürümü
	Build   = "2025-08-27" // tarih veya build ID

	// i18n: Desteklenen diller
	LangEN = "en" // English
	LangTR = "tr" // Türkçe
	LangES = "es" // Español
	LangZH = "zh" // 中文

	// Proje sloganı (i18n ile değiştirilebilir)
	Slogan = "Yapılmayanı yapmak."

	// Explorer & UI temaları
	ThemeDark  = "dark"
	ThemeLight = "light"

	// NFT metadata JSON API base URL
	NFTBaseURL = "https://nft.quantumcoin.org/metadata/"

	// Stake süresi sabitleri (gün)
	StakeShortTerm = 90  // 3 ay
	StakeMidTerm   = 180 // 6 ay
	StakeLongTerm  = 365 // 1 yıl
)
