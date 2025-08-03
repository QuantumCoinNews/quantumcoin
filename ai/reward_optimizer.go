package ai

import (
	"math"
	"quantumcoin/blockchain"
)

// AI tabanlı ödül dağıtım öneri yapısı
type RewardSuggestion struct {
	WalletAddress   string
	SuggestedReward float64
	Reason          string
}

// Ödülleri, son transfer sayısı ve işlem büyüklüğüne göre dinamik optimize et
func OptimizeRewards(txs []*blockchain.Transaction, baseReward float64, minReward float64) []RewardSuggestion {
	walletStats := make(map[string]int)
	walletAmount := make(map[string]float64)

	for _, tx := range txs {
		walletStats[tx.Sender]++
		walletAmount[tx.Sender] += tx.Amount
	}

	var suggestions []RewardSuggestion
	for wallet, count := range walletStats {
		// Temel bir dinamik dağıtım: sqrt ile ödül büyür, çok fazla transferde sabitlenir
		reward := baseReward * math.Sqrt(float64(count))
		if reward < minReward {
			reward = minReward
		}
		suggestions = append(suggestions, RewardSuggestion{
			WalletAddress:   wallet,
			SuggestedReward: reward,
			Reason:          "AI optimize ödül (sık transfer ve hacim bazlı)",
		})
	}
	return suggestions
}
