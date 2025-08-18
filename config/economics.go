// config/economics.go
package config

// Paylaşım yüzdeleri (sabit politika).
// Toplamı 70 + 10 + 10 + 5 = 95; kalan %5 community’ye gider.
// Not: Community yüzdesini ayrıca sabit olarak da tutuyoruz
// ki eski kodlar doğrudan kullanabilsin.
const (
	ShareMiningPct    = 70
	ShareStakingPct   = 10
	ShareDevPct       = 10
	ShareBurnPct      = 5
	ShareCommunityPct = 5 // 100 - (Mining+Staking+Dev+Burn)
)
