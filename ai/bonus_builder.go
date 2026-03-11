package ai

// BuildAIBonuses:
//   - Bloktan veya mempool'dan gelen TxLite listesine göre
//     küçük, hedefli AI bonus önerileri üretir.
//   - Gerçek dağıtım, node tarafında internal.GiveBonus ile yapılır.
func BuildAIBonuses(txs []TxLite) []BonusSuggestion {
	if !Enabled() {
		return nil
	}

	// Önce tüm tx'ler için anomalileri çıkar
	anoms := AnalyzeTransactions(txs)
	if len(anoms) == 0 {
		return nil
	}

	out := make([]BonusSuggestion, 0, len(anoms))

	for _, a := range anoms {
		if a.WalletAddress == "" {
			continue
		}

		// Ortak reason normalizer (reward_optimizer.go içindeki helper)
		reason := normalizeBonusReason(a.Reason)
		if reason == "" {
			reason = "AI bonus"
		}

		// Skora göre yumuşak katmanlı bonus
		amount := 2 // default
		if a.Score > 0.90 {
			amount = 4
		} else if a.Score > 0.70 {
			amount = 3
		}

		out = append(out, BonusSuggestion{
			WalletAddress: a.WalletAddress,
			Source:        "AI",
			Amount:        amount,
			Reason:        reason,
		})
	}

	return out
}
