// Package security contains QuantumCoin project code for this module.
package security

import "sort"

const MinerSafetyGuardReviewVersion = "MINER_SAFETY_GUARD_REVIEW_V1"

type MinerSafetyObservation struct {
	Subject               string `json:"subject"`
	Mode                  string `json:"mode"`
	LoopActive            bool   `json:"loopActive"`
	StopFlagPresent       bool   `json:"stopFlagPresent"`
	DelayMs               int    `json:"delayMs"`
	DuplicateMinerCount   int    `json:"duplicateMinerCount"`
	APIMissingCount       int    `json:"apiMissingCount"`
	MineErrorCount        int    `json:"mineErrorCount"`
	ConsecutiveErrorCount int    `json:"consecutiveErrorCount"`
	MissingAddressCount   int    `json:"missingAddressCount"`
	UnexpectedExitCount   int    `json:"unexpectedExitCount"`
	PanicOrCrashCount     int    `json:"panicOrCrashCount"`
	ManualStopObserved    bool   `json:"manualStopObserved"`
	APIStartAttempted     bool   `json:"apiStartAttempted"`
	VisibleCMDMode        bool   `json:"visibleCmdMode"`
	StableReleaseScript   bool   `json:"stableReleaseScript"`
}

type MinerSafetyFinding struct {
	Subject  string         `json:"subject"`
	Risk     string         `json:"risk"`
	Score    int            `json:"score"`
	Reasons  []string       `json:"reasons"`
	Counters map[string]int `json:"counters"`
}

func AnalyzeMinerSafety(observations []MinerSafetyObservation) map[string]any {
	findings := make([]MinerSafetyFinding, 0)

	for _, obs := range observations {
		subject := obs.Subject
		if subject == "" {
			if obs.Mode != "" {
				subject = obs.Mode
			} else {
				subject = "UNKNOWN_MINER"
			}
		}

		score := 0
		reasons := []string{}

		if obs.DuplicateMinerCount > 0 {
			score += obs.DuplicateMinerCount * 35
			reasons = append(reasons, "duplicate miner process/window signal observed")
		}

		if obs.APIMissingCount > 0 {
			score += obs.APIMissingCount * 20
			reasons = append(reasons, "miner loop observed API missing/unreachable")
		}

		if obs.MineErrorCount > 0 {
			score += obs.MineErrorCount * 12
			reasons = append(reasons, "mine endpoint errors observed")
		}

		if obs.ConsecutiveErrorCount >= 10 {
			score += 45
			reasons = append(reasons, "high consecutive miner loop error count")
		} else if obs.ConsecutiveErrorCount >= 3 {
			score += 20
			reasons = append(reasons, "repeated miner loop errors observed")
		}

		if obs.MissingAddressCount > 0 {
			score += obs.MissingAddressCount * 25
			reasons = append(reasons, "missing miner address observed")
		}

		if obs.UnexpectedExitCount > 0 {
			score += obs.UnexpectedExitCount * 25
			reasons = append(reasons, "unexpected miner process exit observed")
		}

		if obs.PanicOrCrashCount > 0 {
			score += obs.PanicOrCrashCount * 35
			reasons = append(reasons, "miner panic/crash signal observed")
		}

		if obs.LoopActive && !obs.StopFlagPresent && obs.DelayMs <= 0 {
			score += 20
			reasons = append(reasons, "active miner loop has no visible delay configuration")
		}

		if obs.LoopActive && obs.DelayMs > 0 && obs.DelayMs < 500 {
			score += 15
			reasons = append(reasons, "miner loop delay is very low")
		}

		if obs.LoopActive && !obs.StableReleaseScript {
			score += 15
			reasons = append(reasons, "miner loop is not marked as stable release script")
		}

		if obs.ManualStopObserved && score > 0 {
			reasons = append(reasons, "manual stop path exists but warnings still require review")
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
			findings = append(findings, MinerSafetyFinding{
				Subject: subject,
				Risk:    risk,
				Score:   score,
				Reasons: reasons,
				Counters: map[string]int{
					"delayMs":               obs.DelayMs,
					"duplicateMinerCount":   obs.DuplicateMinerCount,
					"apiMissingCount":       obs.APIMissingCount,
					"mineErrorCount":        obs.MineErrorCount,
					"consecutiveErrorCount": obs.ConsecutiveErrorCount,
					"missingAddressCount":   obs.MissingAddressCount,
					"unexpectedExitCount":   obs.UnexpectedExitCount,
					"panicOrCrashCount":     obs.PanicOrCrashCount,
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
		"version":              MinerSafetyGuardReviewVersion,
		"mode":                 "READ_ONLY_MINER_SAFETY_REVIEW",
		"overallRisk":          overallRisk,
		"observationsAnalyzed": len(observations),
		"findings":             findings,
		"policyNote":           "This review only reports miner loop/process safety risk for admin/AI/security review. It does not start mining, stop mining, kill processes, change mining rewards, change balances, write to chain, replace chain, or change consensus.",
		"safeBoundary": map[string]bool{
			"readOnly":              true,
			"startsMining":          false,
			"stopsMining":           false,
			"killsProcess":          false,
			"changesMiningReward":   false,
			"changesMempool":        false,
			"changesChain":          false,
			"replacesChain":         false,
			"changesBalances":       false,
			"changesConsensus":      false,
			"changesTotalSupply":    false,
			"performsTransfer":      false,
			"modifiesReleaseScript": false,
		},
	}
}
