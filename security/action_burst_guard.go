// Package security contains QuantumCoin project code for this module.
package security

import "sort"

const ActionBurstGuardReviewVersion = "ACTION_BURST_GUARD_REVIEW_V1"

type ActionBurstObservation struct {
	Subject            string `json:"subject"`
	ClientIP           string `json:"clientIp"`
	PlayerID           string `json:"playerId"`
	Endpoint           string `json:"endpoint"`
	WindowSeconds      int    `json:"windowSeconds"`
	RequestCount       int    `json:"requestCount"`
	RateLimitedCount   int    `json:"rateLimitedCount"`
	Status429Count     int    `json:"status429Count"`
	BlockedCount       int    `json:"blockedCount"`
	CooldownHitCount   int    `json:"cooldownHitCount"`
	DailyLimitHitCount int    `json:"dailyLimitHitCount"`
	SensitiveEndpoint  bool   `json:"sensitiveEndpoint"`
	P2PFloodCount      int    `json:"p2pFloodCount"`
}

type ActionBurstFinding struct {
	Subject  string         `json:"subject"`
	Risk     string         `json:"risk"`
	Score    int            `json:"score"`
	Reasons  []string       `json:"reasons"`
	Counters map[string]int `json:"counters"`
}

func AnalyzeActionBurst(observations []ActionBurstObservation) map[string]any {
	findings := make([]ActionBurstFinding, 0)

	for _, obs := range observations {
		subject := obs.Subject
		if subject == "" {
			if obs.PlayerID != "" {
				subject = obs.PlayerID
			} else if obs.ClientIP != "" {
				subject = obs.ClientIP
			} else {
				subject = "UNKNOWN_ACTOR"
			}
		}

		score := 0
		reasons := []string{}

		if obs.RequestCount >= 200 {
			score += 60
			reasons = append(reasons, "very high request count in action window")
		} else if obs.RequestCount >= 60 {
			score += 35
			reasons = append(reasons, "high request count in action window")
		} else if obs.RequestCount >= 20 {
			score += 15
			reasons = append(reasons, "elevated request count in action window")
		}

		if obs.RateLimitedCount > 0 {
			score += obs.RateLimitedCount * 15
			reasons = append(reasons, "rate-limited requests observed")
		}

		if obs.Status429Count > 0 {
			score += obs.Status429Count * 12
			reasons = append(reasons, "HTTP 429 responses observed")
		}

		if obs.BlockedCount > 0 {
			score += obs.BlockedCount * 20
			reasons = append(reasons, "action burst blocks observed")
		}

		if obs.CooldownHitCount > 0 {
			score += obs.CooldownHitCount * 8
			reasons = append(reasons, "cooldown hits observed")
		}

		if obs.DailyLimitHitCount > 0 {
			score += obs.DailyLimitHitCount * 8
			reasons = append(reasons, "daily limit hits observed")
		}

		if obs.SensitiveEndpoint && obs.RequestCount >= 10 {
			score += 20
			reasons = append(reasons, "sensitive endpoint burst observed")
		}

		if obs.P2PFloodCount > 0 {
			score += obs.P2PFloodCount * 20
			reasons = append(reasons, "P2P flood/rate-limit signal observed")
		}

		risk := "LOW"
		switch {
		case score >= 70:
			risk = "HIGH"
		case score >= 35:
			risk = "MEDIUM"
		case score >= 15:
			risk = "WATCH"
		}

		if score > 0 {
			findings = append(findings, ActionBurstFinding{
				Subject: subject,
				Risk:    risk,
				Score:   score,
				Reasons: reasons,
				Counters: map[string]int{
					"windowSeconds":      obs.WindowSeconds,
					"requestCount":       obs.RequestCount,
					"rateLimitedCount":   obs.RateLimitedCount,
					"status429Count":     obs.Status429Count,
					"blockedCount":       obs.BlockedCount,
					"cooldownHitCount":   obs.CooldownHitCount,
					"dailyLimitHitCount": obs.DailyLimitHitCount,
					"p2pFloodCount":      obs.P2PFloodCount,
				},
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Score == findings[j].Score {
			return findings[i].Subject < findings[j].Subject
		}
		return findings[i].Score > findings[j].Score
	})

	overallRisk := "LOW"
	if len(findings) > 0 {
		switch {
		case findings[0].Score >= 70:
			overallRisk = "HIGH"
		case findings[0].Score >= 35:
			overallRisk = "MEDIUM"
		case findings[0].Score >= 15:
			overallRisk = "WATCH"
		}
	}

	return map[string]any{
		"version":              ActionBurstGuardReviewVersion,
		"mode":                 "READ_ONLY_ACTION_BURST_RATE_LIMIT_REVIEW",
		"overallRisk":          overallRisk,
		"observationsAnalyzed": len(observations),
		"findings":             findings,
		"policyNote":           "This review only reports API/TG/P2P rate-limit and action-burst risk for admin/AI/security review. It does not block requests, ban clients, change rate limits, mutate mempool, write to chain, transfer funds, or start mining.",
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"blocksRequest":      false,
			"bansClient":         false,
			"changesRateLimits":  false,
			"changesMempool":     false,
			"changesChain":       false,
			"replacesChain":      false,
			"changesBalances":    false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
			"performsTransfer":   false,
			"startsMining":       false,
		},
	}
}
