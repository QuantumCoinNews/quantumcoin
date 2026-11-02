// Package security contains QuantumCoin project code for this module.
package security

import (
	"sort"
	"strings"
)

const IPAPIReputationVersion = "IP_API_REPUTATION_V1"

type IPAPIReputationObservation struct {
	ClientIP    string `json:"clientIp"`
	Endpoint    string `json:"endpoint"`
	Method      string `json:"method"`
	StatusCode  int    `json:"statusCode"`
	RateLimited bool   `json:"rateLimited"`
	AuthFailed  bool   `json:"authFailed"`
	Blocked     bool   `json:"blocked"`
}

type IPAPIReputationFinding struct {
	Subject  string         `json:"subject"`
	Risk     string         `json:"risk"`
	Score    int            `json:"score"`
	Reasons  []string       `json:"reasons"`
	Counters map[string]int `json:"counters"`
}

type ipAPIReputationCounter struct {
	subject       string
	requestCount  int
	endpointSet   map[string]bool
	rateLimited   int
	authFailed    int
	blocked       int
	clientErrors  int
	serverErrors  int
	sensitiveHits int
	emptyClient   int
}

func isSensitiveAPIEndpointV1(endpoint string) bool {
	e := strings.ToLower(strings.TrimSpace(endpoint))
	if e == "" {
		return false
	}

	sensitive := []string{
		"/api/tx/send",
		"/api/mine",
		"/api/v1/withdraw/request",
		"/api/settlement",
		"/api/watch/claim",
		"/api/social/claim",
		"/api/admin",
	}

	for _, marker := range sensitive {
		if strings.Contains(e, marker) {
			return true
		}
	}

	return false
}

func AnalyzeIPAPIReputation(observations []IPAPIReputationObservation) map[string]any {
	counters := map[string]*ipAPIReputationCounter{}

	for _, obs := range observations {
		subject := strings.TrimSpace(obs.ClientIP)
		emptyClient := false
		if subject == "" {
			subject = "UNKNOWN_CLIENT"
			emptyClient = true
		}

		c, ok := counters[subject]
		if !ok {
			c = &ipAPIReputationCounter{
				subject:     subject,
				endpointSet: map[string]bool{},
			}
			counters[subject] = c
		}

		c.requestCount++

		endpoint := strings.ToLower(strings.TrimSpace(obs.Endpoint))
		if endpoint != "" {
			c.endpointSet[endpoint] = true
		}

		if obs.RateLimited || obs.StatusCode == 429 {
			c.rateLimited++
		}

		if obs.AuthFailed || obs.StatusCode == 401 || obs.StatusCode == 403 {
			c.authFailed++
		}

		if obs.Blocked {
			c.blocked++
		}

		if obs.StatusCode >= 400 && obs.StatusCode < 500 {
			c.clientErrors++
		}

		if obs.StatusCode >= 500 {
			c.serverErrors++
		}

		if isSensitiveAPIEndpointV1(endpoint) {
			c.sensitiveHits++
		}

		if emptyClient {
			c.emptyClient++
		}
	}

	findings := make([]IPAPIReputationFinding, 0)

	for subject, c := range counters {
		score := 0
		reasons := []string{}
		endpointSpread := len(c.endpointSet)

		if c.requestCount >= 100 {
			score += 35
			reasons = append(reasons, "high request count from same client")
		} else if c.requestCount >= 30 {
			score += 18
			reasons = append(reasons, "elevated request count from same client")
		}

		if endpointSpread >= 15 {
			score += 20
			reasons = append(reasons, "wide endpoint spread from same client")
		} else if endpointSpread >= 7 {
			score += 10
			reasons = append(reasons, "moderate endpoint spread from same client")
		}

		if c.rateLimited > 0 {
			score += 15 * c.rateLimited
			reasons = append(reasons, "rate-limited API requests observed")
		}

		if c.authFailed > 0 {
			score += 8 * c.authFailed
			reasons = append(reasons, "authentication or authorization failures observed")
		}

		if c.blocked > 0 {
			score += 20 * c.blocked
			reasons = append(reasons, "blocked API requests observed")
		}

		if c.sensitiveHits >= 25 {
			score += 20
			reasons = append(reasons, "high sensitive endpoint access count")
		} else if c.sensitiveHits >= 10 {
			score += 10
			reasons = append(reasons, "sensitive endpoint access observed")
		}

		if c.emptyClient > 0 {
			score += 10 * c.emptyClient
			reasons = append(reasons, "missing client IP observed")
		}

		if c.clientErrors >= 50 {
			score += 10
			reasons = append(reasons, "high client error count")
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
			findings = append(findings, IPAPIReputationFinding{
				Subject: subject,
				Risk:    risk,
				Score:   score,
				Reasons: reasons,
				Counters: map[string]int{
					"requestCount":     c.requestCount,
					"endpointSpread":   endpointSpread,
					"rateLimited":      c.rateLimited,
					"authFailed":       c.authFailed,
					"blocked":          c.blocked,
					"clientErrors":     c.clientErrors,
					"serverErrors":     c.serverErrors,
					"sensitiveHits":    c.sensitiveHits,
					"emptyClientCount": c.emptyClient,
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
		"version":              IPAPIReputationVersion,
		"mode":                 "READ_ONLY_REPUTATION_ANALYSIS",
		"overallRisk":          overallRisk,
		"observationsAnalyzed": len(observations),
		"findings":             findings,
		"rules": []string{
			"high request count per client",
			"wide endpoint spread per client",
			"rate-limited API requests",
			"auth failures",
			"blocked requests",
			"sensitive endpoint access",
			"missing client IP",
			"high client error count",
		},
		"policyNote": "This analyzer only reports IP/API reputation risk. It does not ban clients, block requests, change rate limits, mutate mempool, or write to chain.",
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"bansClient":         false,
			"blocksRequest":      false,
			"changesRateLimits":  false,
			"changesMempool":     false,
			"changesChain":       false,
			"changesBalances":    false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
			"performsTransfer":   false,
			"startsMining":       false,
		},
	}
}
