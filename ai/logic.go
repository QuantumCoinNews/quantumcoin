package ai

import "time"

// RecentTxs filters: verilen TxLite listesinden son N saat içindekileri alır.
func RecentTxs(txs []TxLite, hours int) []TxLite {
	if hours <= 0 {
		return txs
	}
	cut := time.Now().Add(-time.Duration(hours) * time.Hour)

	out := make([]TxLite, 0, len(txs))
	for _, t := range txs {
		if t.Timestamp.After(cut) {
			out = append(out, t)
		}
	}
	return out
}

// GroupAnomaliesByWallet aynı cüzdandan gelen anomaly'leri birleştirir.
// Bu GUI'de tek satır göstermek için işe yarar.
func GroupAnomaliesByWallet(anoms []AnomalyReport) map[string][]AnomalyReport {
	m := make(map[string][]AnomalyReport)
	for _, a := range anoms {
		m[a.WalletAddress] = append(m[a.WalletAddress], a)
	}
	return m
}

// PickTopAnomalies score'a göre en iyi K anomaly'i döndürür.
func PickTopAnomalies(anoms []AnomalyReport, k int) []AnomalyReport {
	if k <= 0 || len(anoms) <= k {
		return anoms
	}

	// çok basit selection; sort etmek de olur
	out := make([]AnomalyReport, 0, k)
	tmp := make([]AnomalyReport, len(anoms))
	copy(tmp, anoms)

	// küçük k'lar için basit O(k*n) yeter
	for i := 0; i < k; i++ {
		bestIdx := -1
		for j := i; j < len(tmp); j++ {
			if bestIdx == -1 || tmp[j].Score > tmp[bestIdx].Score {
				bestIdx = j
			}
		}
		if bestIdx == -1 {
			break
		}
		out = append(out, tmp[bestIdx])
		// seçileni sona at
		tmp[i], tmp[bestIdx] = tmp[bestIdx], tmp[i]
	}
	return out
}
