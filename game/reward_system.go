// Package game contains QuantumCoin project code for this module.
package game

import "fmt"

// Eşikler (gerekirse config'e taşınabilir)
const (
	RewardThresholdBronze = 500
	RewardThresholdSilver = 1000
	RewardThresholdGold   = 2000
)

// GiveReward documents the QuantumCoin component used by this module.
// Basit ödül yazdırma (zincir içi bonus entegrasyonu yoksa sorun çıkarmaz)
func GiveReward(player string, rewardType string, amount int) {
	if player == "" || amount <= 0 {
		return
	}
	fmt.Printf("[REWARD] %s -> %d x %s\n", player, amount, rewardType)
}

// EvaluateAndReward documents the QuantumCoin component used by this module.
// Skora göre ödül mantığı (yalnızca çağrıldığında işler)
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
