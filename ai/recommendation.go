package ai

import (
	"sort"
	"time"

	"quantumcoin/blockchain"
)

// Kullanıcıya özel öneri yapısı
type WalletRecommendation struct {
	WalletAddress  string
	Recommendation string
	Score          float64 // 0..1
}

// Temel: Aktif olmayanlara "aktif ol", sık transfer yapanlara "ödül avcısı", çok küçük & sık olana "batching" gibi öneriler üret.
// activePeriodDays: "aktif değil" eşiği. transferThreshold: "sık" eşiği.
func GenerateRecommendations(txs []*blockchain.Transaction, activePeriodDays int, transferThreshold int) []WalletRecommendation {
	now := time.Now()
	cutoff := now.Add(-time.Duration(activePeriodDays) * 24 * time.Hour)

	type agg struct {
		last   time.Time
		count  int
		sum    float64
		maxAmt float64
	}
	stats := make(map[string]*agg)

	for _, tx := range txs {
		if tx == nil {
			continue
		}
		a := stats[tx.Sender]
		if a == nil {
			a = &agg{}
			stats[tx.Sender] = a
		}
		a.count++
		a.sum += tx.Amount
		if tx.Amount > a.maxAmt {
			a.maxAmt = tx.Amount
		}
		if tx.Timestamp.After(a.last) {
			a.last = tx.Timestamp
		}
	}

	var recs []WalletRecommendation
	for addr, a := range stats {
		// varsayılan skor/mesaj
		rec := WalletRecommendation{
			WalletAddress:  addr,
			Recommendation: "Normal aktivite.",
			Score:          0.5,
		}

		// inaktif ise
		if a.last.Before(cutoff) {
			rec.Recommendation = "Cüzdan uzun süredir aktif değil, bonus ödüller için tekrar aktif olun!"
			rec.Score = 0.8
		} else if a.count > transferThreshold && a.maxAmt < 5 { // çok sayıda ufak işlem: batching
			rec.Recommendation = "Çok sayıda küçük işlem tespit edildi: mümkünse toplu gönderim (batching) düşün."
			rec.Score = 0.85
		} else if a.count <= 2 && a.maxAmt >= 100 { // nadiren ama büyük
			rec.Recommendation = "Nadiren ama büyük miktarlar: tek işlemi bölerek gönderme ve onay süresini takip et."
			rec.Score = 0.8
		} else if a.count > transferThreshold {
			rec.Recommendation = "Sık transfer yapan cüzdan - ödül avcısı kategorisine eklenebilir."
			rec.Score = 0.7
		}

		recs = append(recs, rec)
	}

	// En anlamlı öneriler üstte
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].Score == recs[j].Score {
			return recs[i].WalletAddress < recs[j].WalletAddress
		}
		return recs[i].Score > recs[j].Score
	})
	return recs
}
