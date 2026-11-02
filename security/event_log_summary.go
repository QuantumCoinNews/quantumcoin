// Package security contains QuantumCoin project code for this module.
package security

const SecurityEventSummaryVersion = "SECURITY_EVENT_SUMMARY_V1"

func SummarizeEvents(events []SecurityEvent) map[string]any {
	bySeverity := map[string]int{}

	for _, ev := range events {
		bySeverity[string(ev.Severity)]++
	}

	overallRisk := "LOW"
	if bySeverity[string(SeverityCritical)] > 0 {
		overallRisk = "CRITICAL"
	} else if bySeverity[string(SeverityHigh)] > 0 {
		overallRisk = "HIGH"
	} else if bySeverity[string(SeverityWarning)] > 0 {
		overallRisk = "WARNING"
	}

	return map[string]any{
		"version":            SecurityEventLogVersion,
		"summaryVersion":     SecurityEventSummaryVersion,
		"overallRisk":        overallRisk,
		"totalEvents":        len(events),
		"bySeverity":         bySeverity,
		"policyNote":         "Security event summaries are local advisory/read-only monitoring output only. They do not block requests, ban clients, change rate limits, change mempool, change chain, replace chain, change balances, change consensus, change total supply, perform transfers, start mining, stop mining, kill processes, or modify release executables.",
		"readOnly":           true,
		"advisoryOnly":       true,
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
		"stopsMining":        false,
		"killsProcess":       false,
	}
}
