package internal

// Bu dosya sadece uyumluluk amaçlıdır. Asıl işlevler bonus_compat.go’dadır.

func InitRewardSystem(path string) { SetBonusFile(path) }

func Reward(address, bonusType string, amount int, title, meta string) {
	GiveBonus(address, bonusType, amount, title, meta)
}
