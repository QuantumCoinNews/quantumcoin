package game

import (
	"fmt"
	"quantumcoin/internal"
)

// Basit eşik kuralları (dilersen config’e taşıyabilirsin)
const (
	RewardThresholdBronze = 500
	RewardThresholdSilver = 1000
	RewardThresholdGold   = 2000
)

// Oyuncuya zincir üstü ödül/bonus verme fonksiyonu
func GiveReward(player string, rewardType string, amount int) {
	if player == "" || amount <= 0 {
		return
	}
	fmt.Printf("%s kullanıcısına %d adet '%s' ödülü verildi.\n", player, amount, rewardType)
	internal.GiveBonus(player, internal.BonusTypeEvent, amount, rewardType, "")
}

// EvaluateAndReward: mevcut skora göre eşik bazlı ödüller
func EvaluateAndReward(gs *GameState, player string) {
	if gs == nil || player == "" {
		return
	}
	score := gs.GetScore(player)
	switch {
	case score >= RewardThresholdGold:
		GiveReward(player, "GoldMilestone", 20)
	case score >= RewardThresholdSilver:
		GiveReward(player, "SilverMilestone", 10)
	case score >= RewardThresholdBronze:
		GiveReward(player, "BronzeMilestone", 5)
	}
}
