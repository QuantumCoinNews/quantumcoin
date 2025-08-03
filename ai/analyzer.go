package ai

import (
	"quantumcoin/blockchain"
	"time"
)

// Şüpheli cüzdan ve işlem analizi için temel veri yapısı
type AnomalyReport struct {
	WalletAddress string
	Suspicious    bool
	Reason        string
}

// Son X saat içinde threshold'dan fazla transfer yapan cüzdanları bul
func AnalyzeTransactions(txs []*blockchain.Transaction, threshold int, periodHours int) []AnomalyReport {
	walletStats := make(map[string]int)
	cutoff := time.Now().Add(-time.Duration(periodHours) * time.Hour)

	for _, tx := range txs {
		if tx.Timestamp.After(cutoff) {
			walletStats[tx.Sender]++
		}
	}

	var reports []AnomalyReport
	for wallet, count := range walletStats {
		if count > threshold {
			reports = append(reports, AnomalyReport{
				WalletAddress: wallet,
				Suspicious:    true,
				Reason:        "High transfer frequency in period",
			})
		}
	}
	return reports
}
