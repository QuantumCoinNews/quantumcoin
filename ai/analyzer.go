package ai

import (
	"sort"
	"time"
)

// AnalyzeTransactions documents the QuantumCoin component used by this module.
// tek analiz fonksiyonumuz BU olsun.
// diğer dosyalarda tekrar yazmayacağız.
func AnalyzeTransactions(txs []TxLite) []AnomalyReport {
	enabled, level := configSnapshot()
	if !enabled {
		return nil
	}
	amountThreshold, burstCount := BigAmountThreshold, BurstCountThreshold
	switch level {
	case AILevelLow:
		amountThreshold, burstCount = 1000, 10
	case AILevelAggressive:
		amountThreshold, burstCount = 250, 3
	}

	out := make([]AnomalyReport, 0, len(txs))

	// adreslere göre grupla
	perAddr := make(map[string][]TxLite)
	seen := make(map[string]bool)
	for _, t := range txs {
		// Repeated observation of one transaction is not a burst.
		if t.TxID != "" {
			if seen[t.TxID] {
				continue
			}
			seen[t.TxID] = true
		}
		// büyük tutar kuralı
		if t.Amount >= amountThreshold {
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
	addresses := make([]string, 0, len(perAddr))
	for addr := range perAddr {
		addresses = append(addresses, addr)
	}
	sort.Strings(addresses)
	for _, addr := range addresses {
		list := perAddr[addr]
		sort.Slice(list, func(i, j int) bool {
			return list[i].Timestamp.Before(list[j].Timestamp)
		})

		window := time.Duration(BurstWindowSeconds) * time.Second
		left := 0
		for right := range list {
			for list[right].Timestamp.Sub(list[left].Timestamp) > window {
				left++
			}
			if right-left+1 >= burstCount {
				out = append(out, AnomalyReport{
					WalletAddress: addr,
					TxID:          list[left].TxID,
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
