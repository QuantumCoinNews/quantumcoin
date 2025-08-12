package ai

import (
	"sort"
	"time"

	"quantumcoin/blockchain"
)

// Şüpheli cüzdan ve işlem analizi için temel veri yapısı
type AnomalyReport struct {
	WalletAddress string
	Count         int
	AvgAmount     float64
	MaxAmount     float64
	Suspicious    bool
	Reason        string
}

// Son X saat içinde threshold'dan fazla transfer yapan cüzdanları bul.
// Ek heurstik: Maks tutar ortalamanın 2x üstündeyse şüphe derecesini yükselt.
func AnalyzeTransactions(txs []*blockchain.Transaction, threshold int, periodHours int) []AnomalyReport {
	if threshold <= 0 {
		threshold = 5
	}
	cutoff := time.Now().Add(-time.Duration(periodHours) * time.Hour)

	type agg struct {
		count int
		sum   float64
		max   float64
	}
	stats := make(map[string]*agg)

	for _, tx := range txs {
		if tx == nil {
			continue
		}
		if tx.Timestamp.Before(cutoff) {
			continue
		}
		a := stats[tx.Sender]
		if a == nil {
			a = &agg{}
			stats[tx.Sender] = a
		}
		a.count++
		a.sum += tx.Amount
		if tx.Amount > a.max {
			a.max = tx.Amount
		}
	}

	reports := make([]AnomalyReport, 0, len(stats))
	for addr, a := range stats {
		if a.count == 0 {
			continue
		}
		avg := a.sum / float64(a.count)
		susp := a.count > threshold
		reason := "High transfer frequency in period"
		// ek heurstik
		if susp && a.max >= 2*avg {
			reason = "High frequency + amount spikes"
		}
		reports = append(reports, AnomalyReport{
			WalletAddress: addr,
			Count:         a.count,
			AvgAmount:     avg,
			MaxAmount:     a.max,
			Suspicious:    susp,
			Reason:        reason,
		})
	}

	// Deterministik çıktı: cüzdan adresine göre sırala
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].WalletAddress < reports[j].WalletAddress
	})
	return reports
}
