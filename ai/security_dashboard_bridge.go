package ai

const AISecurityDashboardBridgeVersion = "AI_SECURITY_DASHBOARD_BRIDGE_V1"

type AISecurityDashboardBridgeInput struct {
	AdminReport map[string]any `json:"adminReport"`
	Dashboard   map[string]any `json:"dashboard"`
}

func BuildAISecurityDashboardBridge(input AISecurityDashboardBridgeInput) map[string]any {
	adminRisk := aiSecurityBridgeStringFromMap(input.AdminReport, "overallRisk", "UNKNOWN")
	adminStatus := aiSecurityBridgeStringFromMap(input.AdminReport, "overallStatus", "UNKNOWN")
	securityScore := aiSecurityBridgeIntFromMap(input.AdminReport, "securityScore", 0)

	aiPosture := "OBSERVE"
	if adminRisk == "HIGH" {
		aiPosture = "ESCALATE"
	} else if adminRisk == "MEDIUM" {
		aiPosture = "REVIEW_REQUIRED"
	} else if adminRisk == "WATCH" {
		aiPosture = "WATCH"
	} else if adminRisk == "LOW" {
		aiPosture = "NORMAL"
	}

	aiFindings := []string{}

	if adminRisk == "WATCH" {
		aiFindings = append(aiFindings, "Network is in watch mode based on admin security report.")
	}
	if adminRisk == "MEDIUM" || adminRisk == "HIGH" {
		aiFindings = append(aiFindings, "Security review should be performed before public/testnet exposure.")
	}
	if securityScore >= 90 && (adminRisk == "LOW" || adminRisk == "WATCH") {
		aiFindings = append(aiFindings, "Security score is healthy, but contextual network warnings may still require attention.")
	}
	if len(aiFindings) == 0 {
		aiFindings = append(aiFindings, "No immediate AI security finding.")
	}

	return map[string]any{
		"version":         AISecurityDashboardBridgeVersion,
		"mode":            "READ_ONLY_AI_SECURITY_BRIDGE",
		"aiPosture":       aiPosture,
		"adminRisk":       adminRisk,
		"adminStatus":     adminStatus,
		"securityScore":   securityScore,
		"analysisPurpose": "Expose existing security dashboard/admin report data to the AI layer for advisory monitoring only.",
		"aiFindings":      aiFindings,
		"inputs": map[string]any{
			"adminReport": input.AdminReport,
			"dashboard":   input.Dashboard,
		},
		"safeBoundary": map[string]bool{
			"readOnly":              true,
			"aiAdvisoryOnly":        true,
			"changesMempool":        false,
			"changesChain":          false,
			"changesBalances":       false,
			"changesConsensus":      false,
			"changesTotalSupply":    false,
			"changesMiningReward":   false,
			"writesBonus":           false,
			"performsTransfer":      false,
			"startsMining":          false,
			"replacesChain":         false,
			"rejectsBlocks":         false,
			"rejectsTx":             false,
			"autoExecutesDecisions": false,
		},
	}
}

func aiSecurityBridgeStringFromMap(m map[string]any, key string, fallback string) string {
	if m == nil {
		return fallback
	}
	v, ok := m[key]
	if !ok || v == nil {
		return fallback
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fallback
}

func aiSecurityBridgeIntFromMap(m map[string]any, key string, fallback int) int {
	if m == nil {
		return fallback
	}
	v, ok := m[key]
	if !ok || v == nil {
		return fallback
	}

	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return fallback
	}
}
