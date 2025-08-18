// ai/bonuses.go
package ai

import (
	"fmt"
	"quantumcoin/blockchain"
	"quantumcoin/internal"
)

// DistributeAIBonuses: AI analizlerinden bonus dağıtır.
// NOT: Bu dosya ai paketinde; internal bonus_core sadece kayıt yazar ve ai'yi import etmez.
func DistributeAIBonuses(txs []*blockchain.Transaction) {
	// 1) Anomali bonusları
	anomalies := AnalyzeTransactions(txs, 5, 24)
	for _, report := range anomalies {
		if report.Suspicious {
			internal.GiveBonus(
				report.WalletAddress,
				"AI",
				2,
				"AI şüpheli işlem sonrası bilinçlendirme bonusu",
				"",
			)
			fmt.Printf("Anomaly bonus: %+v\n", report)
		}
	}

	// 2) Davranışsal öneri bonusları
	recs := GenerateRecommendations(txs, 14, 10)
	for _, rec := range recs {
		if rec.Score > 0.7 {
			internal.GiveBonus(
				rec.WalletAddress,
				"AI",
				1,
				"AI analizine göre aktiflik/ödül bonusu",
				"",
			)
		}
	}

	// 3) Optimize ödüller
	suggestions := OptimizeRewards(txs, 10, 1)
	for _, sug := range suggestions {
		amt := int(sug.SuggestedReward)
		if amt > 0 {
			internal.GiveBonus(
				sug.WalletAddress,
				"AI",
				amt,
				sug.Reason,
				"",
			)
		}
	}
}
