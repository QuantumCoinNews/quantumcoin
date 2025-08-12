package ai

import (
	"math"
	"sort"

	"quantumcoin/blockchain"
)

// AI tabanlı ödül dağıtım öneri yapısı
type RewardSuggestion struct {
	WalletAddress   string  `json:"wallet_address"`
	SuggestedReward float64 `json:"suggested_reward"`
	Reason          string  `json:"reason"`
}

// Ödülleri, transfer sayısı ve hacme göre dinamik optimize et.
// baseReward: temel katsayı, minReward: alt sınır.
func OptimizeRewards(txs []*blockchain.Transaction, baseReward float64, minReward float64) []RewardSuggestion {
	type agg struct {
		count int
		sum   float64
	}
	byAddr := make(map[string]*agg)

	for _, tx := range txs {
		if tx == nil {
			continue
		}
		a := byAddr[tx.Sender]
		if a == nil {
			a = &agg{}
			byAddr[tx.Sender] = a
		}
		a.count++
		a.sum += tx.Amount
	}

	suggestions := make([]RewardSuggestion, 0, len(byAddr))
	for addr, a := range byAddr {
		if a.count == 0 {
			continue
		}
		// sqrt ile azalan marjinal: çok fazla işlemde artış yavaşlar
		reward := baseReward * math.Sqrt(float64(a.count))
		// hacme küçük ağırlık
		reward += 0.05 * a.sum

		if reward < minReward {
			reward = minReward
		}

		suggestions = append(suggestions, RewardSuggestion{
			WalletAddress:   addr,
			SuggestedReward: reward,
			Reason:          "sıklık (sqrt) ve hacim (5%) heurstikleri",
		})
	}

	// Deterministik: ödüle göre DESC, sonra adrese göre ASC
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].SuggestedReward == suggestions[j].SuggestedReward {
			return suggestions[i].WalletAddress < suggestions[j].WalletAddress
		}
		return suggestions[i].SuggestedReward > suggestions[j].SuggestedReward
	})
	return suggestions
}
