package ai

import (
	"sort"
	"time"
)

// tek analiz fonksiyonumuz BU olsun.
// diğer dosyalarda tekrar yazmayacağız.
func AnalyzeTransactions(txs []TxLite) []AnomalyReport {
	if !Enabled() {
		return nil
	}

	out := make([]AnomalyReport, 0, len(txs))

	// adreslere göre grupla
	perAddr := make(map[string][]TxLite)
	for _, t := range txs {
		// büyük tutar kuralı
		if t.Amount >= BigAmountThreshold {
			out = append(out, AnomalyReport{
				WalletAddress: t.WalletAddress,
				TxID:          t.TxID,
				Suspicious:    true,
				Score:         0.7,
				Reason:        "large transaction",
			})
		}
		perAddr[t.WalletAddress] = append(perAddr[t.WalletAddress], t)
	}

	// frekans kuralı
	for addr, list := range perAddr {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Timestamp.Before(list[j].Timestamp)
		})

		window := time.Duration(BurstWindowSeconds) * time.Second
		for i := 0; i < len(list); i++ {
			count := 1
			for j := i + 1; j < len(list); j++ {
				if list[j].Timestamp.Sub(list[i].Timestamp) <= window {
					count++
				} else {
					break
				}
			}
			if count >= BurstCountThreshold {
				out = append(out, AnomalyReport{
					WalletAddress: addr,
					TxID:          list[i].TxID,
					Suspicious:    true,
					Score:         0.6,
					Reason:        "high frequency activity",
				})
				break
			}
		}
	}

	return out
}
