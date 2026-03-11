// ai/recommendation.go
package ai

// anomaly listesinden kullanıcıya gösterilebilir öneri üret
func BuildWalletRecommendations(anoms []AnomalyReport) []Recommendation {
	out := make([]Recommendation, 0, len(anoms))

	for _, a := range anoms {
		action := "monitor"
		msg := "AI flagged activity"
		score := a.Score
		if score <= 0 {
			score = 0.5
		}

		// biraz yüksek skorda daha ciddi dille konuş
		if a.Score >= 0.65 {
			action = "educate-user"
			msg = "High-risk pattern detected"
		}

		out = append(out, Recommendation{
			WalletAddress: a.WalletAddress,
			Action:        action,
			Reason:        a.Reason,
			Message:       msg,
			Score:         score,
		})
	}

	return out
}
