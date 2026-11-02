// Package security contains QuantumCoin project code for this module.
package security

import "sort"

const GUICLIAPIGuardrailsReviewVersion = "GUI_CLI_API_GUARDRAILS_REVIEW_V1"

type GUICLIAPIGuardrailObservation struct {
	Subject                 string `json:"subject"`
	Surface                 string `json:"surface"`
	APIBase                 string `json:"apiBase"`
	Endpoint                string `json:"endpoint"`
	UsesLocalhost           bool   `json:"usesLocalhost"`
	UsesUnexpectedPort      bool   `json:"usesUnexpectedPort"`
	UsesAPIBalance          bool   `json:"usesApiBalance"`
	UsesBalanceCache        bool   `json:"usesBalanceCache"`
	UsesMinedBalanceLocal   bool   `json:"usesMinedBalanceLocal"`
	UsesBonusStoreLocal     bool   `json:"usesBonusStoreLocal"`
	UsesTxSendEndpoint      bool   `json:"usesTxSendEndpoint"`
	UsesSendFallback        bool   `json:"usesSendFallback"`
	StartsMinerViaAPI       bool   `json:"startsMinerViaApi"`
	StopsMinerViaAPI        bool   `json:"stopsMinerViaApi"`
	StartsProcess           bool   `json:"startsProcess"`
	KillsProcess            bool   `json:"killsProcess"`
	MissingValidationCount  int    `json:"missingValidationCount"`
	EndpointFallbackCount   int    `json:"endpointFallbackCount"`
	CriticalEndpoint        bool   `json:"criticalEndpoint"`
	TransferEndpoint        bool   `json:"transferEndpoint"`
	MiningEndpoint          bool   `json:"miningEndpoint"`
	TimeoutOrAPIErrorCount  int    `json:"timeoutOrApiErrorCount"`
	StaleCacheFallbackCount int    `json:"staleCacheFallbackCount"`
}

type GUICLIAPIGuardrailFinding struct {
	Subject  string         `json:"subject"`
	Risk     string         `json:"risk"`
	Score    int            `json:"score"`
	Reasons  []string       `json:"reasons"`
	Counters map[string]int `json:"counters"`
}

func AnalyzeGUICLIAPIGuardrails(observations []GUICLIAPIGuardrailObservation) map[string]any {
	findings := make([]GUICLIAPIGuardrailFinding, 0)

	for _, obs := range observations {
		subject := obs.Subject
		if subject == "" {
			if obs.Surface != "" {
				subject = obs.Surface
			} else if obs.Endpoint != "" {
				subject = obs.Endpoint
			} else {
				subject = "UNKNOWN_GUI_CLI_API_SURFACE"
			}
		}

		score := 0
		reasons := []string{}

		if obs.UsesUnexpectedPort {
			score += 25
			reasons = append(reasons, "GUI/CLI/API surface uses unexpected API port")
		}
		if !obs.UsesLocalhost && obs.APIBase != "" {
			score += 20
			reasons = append(reasons, "GUI/CLI/API surface is not explicitly local-only")
		}
		if obs.UsesBalanceCache {
			score += 35
			reasons = append(reasons, "balance cache usage observed on GUI/API path")
		}
		if obs.UsesMinedBalanceLocal {
			score += 35
			reasons = append(reasons, "local mined_balance usage observed on GUI/API path")
		}
		if obs.UsesBonusStoreLocal {
			score += 25
			reasons = append(reasons, "local bonus_store usage observed on GUI/API path")
		}
		if obs.StaleCacheFallbackCount > 0 {
			score += obs.StaleCacheFallbackCount * 15
			reasons = append(reasons, "stale/local cache fallback signal observed")
		}
		if obs.MissingValidationCount > 0 {
			score += obs.MissingValidationCount * 15
			reasons = append(reasons, "missing validation guard signal observed")
		}
		if obs.EndpointFallbackCount > 0 {
			score += obs.EndpointFallbackCount * 10
			reasons = append(reasons, "endpoint fallback path requires review")
		}
		if obs.CriticalEndpoint {
			score += 15
			reasons = append(reasons, "critical GUI/CLI/API endpoint requires guardrail review")
		}
		if obs.TransferEndpoint && !obs.UsesTxSendEndpoint {
			score += 30
			reasons = append(reasons, "transfer path is not using canonical /api/tx/send endpoint")
		}
		if obs.TransferEndpoint && obs.UsesSendFallback {
			score += 15
			reasons = append(reasons, "transfer endpoint fallback observed")
		}
		if obs.MiningEndpoint && obs.StartsMinerViaAPI {
			score += 15
			reasons = append(reasons, "miner start endpoint is exposed and requires operator guardrails")
		}
		if obs.StartsProcess {
			score += 15
			reasons = append(reasons, "GUI/CLI/API path may start a local process")
		}
		if obs.KillsProcess {
			score += 30
			reasons = append(reasons, "GUI/CLI/API path may kill a local process")
		}
		if obs.TimeoutOrAPIErrorCount > 0 {
			score += obs.TimeoutOrAPIErrorCount * 8
			reasons = append(reasons, "API timeout/error handling signal observed")
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
			findings = append(findings, GUICLIAPIGuardrailFinding{
				Subject: subject,
				Risk:    risk,
				Score:   score,
				Reasons: reasons,
				Counters: map[string]int{
					"missingValidationCount":  obs.MissingValidationCount,
					"endpointFallbackCount":   obs.EndpointFallbackCount,
					"timeoutOrApiErrorCount":  obs.TimeoutOrAPIErrorCount,
					"staleCacheFallbackCount": obs.StaleCacheFallbackCount,
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
		"version":              GUICLIAPIGuardrailsReviewVersion,
		"mode":                 "READ_ONLY_GUI_CLI_API_GUARDRAILS_REVIEW",
		"overallRisk":          overallRisk,
		"observationsAnalyzed": len(observations),
		"findings":             findings,
		"policyNote":           "This review only reports GUI/CLI/API guardrail risk for admin/AI/security review. It does not change GUI behavior, change CLI commands, change API routes, block requests, start mining, stop mining, kill processes, transfer funds, change balances, write to chain, replace chain, or change consensus.",
		"safeBoundary": map[string]bool{
			"readOnly":                  true,
			"changesGUI":                false,
			"changesCLI":                false,
			"changesAPI":                false,
			"changesEndpoint":           false,
			"blocksRequest":             false,
			"startsMining":              false,
			"stopsMining":               false,
			"killsProcess":              false,
			"performsTransfer":          false,
			"changesBalances":           false,
			"changesMempool":            false,
			"changesChain":              false,
			"replacesChain":             false,
			"changesConsensus":          false,
			"changesTotalSupply":        false,
			"modifiesReleaseExecutable": false,
		},
	}
}
