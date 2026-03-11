package ai

// OptimizeRewards:
// - /api/ai/analysis içinde "reward_suggestions" alanını doldurmak için kullanılır.
// - Şu anda basitçe BuildAIBonuses(txs) çıktısını döndürüyor.
// - imzada AnomalyReport kullanıyoruz ki AnalyzeTransactions ile tam uyumlu olsun.
func OptimizeRewards(txs []TxLite, anoms []AnomalyReport) []BonusSuggestion {
	if !Enabled() {
		return nil
	}
	if len(txs) == 0 {
		return nil
	}

	// Şimdilik tüm ödül optimizasyonunu tek yerden yönetiyoruz:
	// TxLite listesini kullanarak küçük, hedefli bonus önerileri üret.
	return BuildAIBonuses(txs)
}

// DistributeAIBonusesLite:
// - Eski isimle geriye dönük uyumluluk için ince bir sarmalayıcı.
// - Node, main.go içinden hâlâ bu ismi çağırdığı için burada bırakıyoruz.
func DistributeAIBonusesLite(txs []TxLite) []BonusSuggestion {
	if !Enabled() {
		return nil
	}
	return BuildAIBonuses(txs)
}
