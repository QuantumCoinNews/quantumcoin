package ai

import (
	"quantumcoin/blockchain"
	"time"
)

// Kullanıcıya özel öneri yapısı
type WalletRecommendation struct {
	WalletAddress  string
	Recommendation string
	Score          float64
}

// Temel: Aktif olmayanlara "aktif ol", sık transfer yapanlara "ödül avcısı" gibi öneriler üret
func GenerateRecommendations(txs []*blockchain.Transaction, activePeriodDays int, transferThreshold int) []WalletRecommendation {
	now := time.Now()
	walletActivity := make(map[string]time.Time)
	walletTxCount := make(map[string]int)

	for _, tx := range txs {
		if last, ok := walletActivity[tx.Sender]; !ok || tx.Timestamp.After(last) {
			walletActivity[tx.Sender] = tx.Timestamp
		}
		walletTxCount[tx.Sender]++
	}

	var recs []WalletRecommendation
	for wallet, lastActivity := range walletActivity {
		daysAgo := int(now.Sub(lastActivity).Hours() / 24)
		if daysAgo > activePeriodDays {
			recs = append(recs, WalletRecommendation{
				WalletAddress:  wallet,
				Recommendation: "Cüzdan uzun süredir aktif değil, bonus ödüller için tekrar aktif olun!",
				Score:          0.8,
			})
		} else if walletTxCount[wallet] > transferThreshold {
			recs = append(recs, WalletRecommendation{
				WalletAddress:  wallet,
				Recommendation: "Sık transfer yapan cüzdan - ödül avcısı kategorisine eklenebilir.",
				Score:          0.7,
			})
		} else {
			recs = append(recs, WalletRecommendation{
				WalletAddress:  wallet,
				Recommendation: "Normal aktivite.",
				Score:          0.5,
			})
		}
	}
	return recs
}
