// Package security contains QuantumCoin project code for this module.
package security

const SecurityAdminReadonlyReportVersion = "SECURITY_ADMIN_READONLY_REPORT_V1"

type SecurityAdminReportInput struct {
	Summary             map[string]any `json:"summary"`
	Score               map[string]any `json:"score"`
	NodeHealth          map[string]any `json:"nodeHealth"`
	PeerReputation      map[string]any `json:"peerReputation"`
	MinerSafety         map[string]any `json:"minerSafety"`
	ActionBurstGuard    map[string]any `json:"actionBurstGuard"`
	WalletAnomaly       map[string]any `json:"walletAnomaly"`
	MempoolAbuse        map[string]any `json:"mempoolAbuse"`
	DoubleSpendConflict map[string]any `json:"doubleSpendConflict"`
	AntiSpamFeePolicy   map[string]any `json:"antiSpamFeePolicy"`
	FinalityCheckpoint  map[string]any `json:"finalityCheckpoint"`
}

func BuildSecurityAdminReadonlyReport(input SecurityAdminReportInput) map[string]any {
	securityRisk := stringFromMap(input.Score, "risk", "UNKNOWN")
	securityScore := intFromMap(input.Score, "score", 0)
	nodeHealth := stringFromMap(input.NodeHealth, "health", "UNKNOWN")
	nodeSync := stringFromMap(input.NodeHealth, "syncStatus", "UNKNOWN")
	peerRisk := stringFromMap(input.PeerReputation, "overallRisk", "UNKNOWN")
	actionBurstRisk := stringFromMap(input.ActionBurstGuard, "overallRisk", "UNKNOWN")
	minerSafetyRisk := stringFromMap(input.MinerSafety, "overallRisk", "UNKNOWN")
	walletRisk := stringFromMap(input.WalletAnomaly, "overallRisk", "UNKNOWN")
	mempoolRisk := stringFromMap(input.MempoolAbuse, "overallRisk", "UNKNOWN")
	doubleSpendRisk := stringFromMap(input.DoubleSpendConflict, "overallRisk", "UNKNOWN")
	feeRisk := stringFromMap(input.AntiSpamFeePolicy, "risk", "UNKNOWN")
	finalityRisk := stringFromMap(input.FinalityCheckpoint, "risk", "UNKNOWN")
	finalityStatus := stringFromMap(input.FinalityCheckpoint, "status", "UNKNOWN")

	overallRisk := "LOW"
	if securityRisk == "HIGH" || minerSafetyRisk == "HIGH" || actionBurstRisk == "HIGH" || peerRisk == "HIGH" || mempoolRisk == "HIGH" || doubleSpendRisk == "HIGH" || walletRisk == "HIGH" || feeRisk == "HIGH" || finalityRisk == "HIGH" {
		overallRisk = "HIGH"
	} else if securityRisk == "MEDIUM" || minerSafetyRisk == "MEDIUM" || actionBurstRisk == "MEDIUM" || peerRisk == "MEDIUM" || mempoolRisk == "MEDIUM" || doubleSpendRisk == "MEDIUM" || walletRisk == "MEDIUM" || feeRisk == "MEDIUM" || finalityRisk == "MEDIUM" {
		overallRisk = "MEDIUM"
	} else if securityRisk == "WATCH" || minerSafetyRisk == "WATCH" || actionBurstRisk == "WATCH" || peerRisk == "WATCH" || mempoolRisk == "WATCH" || doubleSpendRisk == "WATCH" || walletRisk == "WATCH" || feeRisk == "WATCH" || finalityRisk == "WATCH" || nodeHealth == "WATCH" || nodeHealth == "ISOLATED" {
		overallRisk = "WATCH"
	}

	overallStatus := "OK"
	if overallRisk == "WATCH" {
		overallStatus = "WATCH"
	}
	if overallRisk == "MEDIUM" {
		overallStatus = "ATTENTION_REQUIRED"
	}
	if overallRisk == "HIGH" {
		overallStatus = "ACTION_REQUIRED"
	}

	recommendedActions := []string{}

	if nodeHealth == "ISOLATED" || nodeSync == "NO_PEERS_LOCAL_ONLY" {
		recommendedActions = append(recommendedActions, "Connect at least one trusted peer before public/testnet use.")
	}

	if peerRisk == "WATCH" || peerRisk == "MEDIUM" || peerRisk == "HIGH" {
		recommendedActions = append(recommendedActions, "Review P2P peer reputation/ban-scoring warnings before public/testnet exposure.")
	}

	if actionBurstRisk == "WATCH" || actionBurstRisk == "MEDIUM" || actionBurstRisk == "HIGH" {
		recommendedActions = append(recommendedActions, "Review API/TG/P2P rate-limit and action-burst warnings before public/testnet exposure.")
	}

	if minerSafetyRisk == "WATCH" || minerSafetyRisk == "MEDIUM" || minerSafetyRisk == "HIGH" {
		recommendedActions = append(recommendedActions, "Review miner loop/process safety warnings before public/testnet exposure.")
	}

	if finalityRisk == "WATCH" || finalityRisk == "MEDIUM" || finalityRisk == "HIGH" {
		recommendedActions = append(recommendedActions, "Wait for more block depth and peer confirmation before treating recent blocks as final.")
	}

	if mempoolRisk == "WATCH" || mempoolRisk == "MEDIUM" || mempoolRisk == "HIGH" {
		recommendedActions = append(recommendedActions, "Review mempool pressure and suspicious pending transaction patterns.")
	}

	if doubleSpendRisk == "WATCH" || doubleSpendRisk == "MEDIUM" || doubleSpendRisk == "HIGH" {
		recommendedActions = append(recommendedActions, "Review double-spend/conflict monitor output before accepting public/testnet exposure.")
	}

	if walletRisk == "WATCH" || walletRisk == "MEDIUM" || walletRisk == "HIGH" {
		recommendedActions = append(recommendedActions, "Review wallet anomaly findings before release/testnet exposure.")
	}

	if feeRisk == "WATCH" || feeRisk == "MEDIUM" || feeRisk == "HIGH" {
		recommendedActions = append(recommendedActions, "Review anti-spam fee policy recommendation before enabling any future enforcement.")
	}

	if len(recommendedActions) == 0 {
		recommendedActions = append(recommendedActions, "No immediate security action required.")
	}

	return map[string]any{
		"version":       SecurityAdminReadonlyReportVersion,
		"mode":          "READ_ONLY_ADMIN_REPORT",
		"overallStatus": overallStatus,
		"overallRisk":   overallRisk,
		"securityScore": securityScore,
		"summary": map[string]any{
			"securityRisk":    securityRisk,
			"nodeHealth":      nodeHealth,
			"nodeSync":        nodeSync,
			"peerRisk":        peerRisk,
			"walletRisk":      walletRisk,
			"mempoolRisk":     mempoolRisk,
			"doubleSpendRisk": doubleSpendRisk,
			"feePolicyRisk":   feeRisk,
			"finalityRisk":    finalityRisk,
			"finalityStatus":  finalityStatus,
		},
		"recommendedActions": recommendedActions,
		"sections": map[string]any{
			"securitySummary":     input.Summary,
			"securityScore":       input.Score,
			"nodeHealth":          input.NodeHealth,
			"peerReputation":      input.PeerReputation,
			"actionBurstGuard":    input.ActionBurstGuard,
			"minerSafety":         input.MinerSafety,
			"walletAnomaly":       input.WalletAnomaly,
			"mempoolAbuse":        input.MempoolAbuse,
			"doubleSpendConflict": input.DoubleSpendConflict,
			"antiSpamFeePolicy":   input.AntiSpamFeePolicy,
			"finalityCheckpoint":  input.FinalityCheckpoint,
		},
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"adminReportOnly":    true,
			"changesMempool":     false,
			"changesChain":       false,
			"changesBalances":    false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
			"performsTransfer":   false,
			"startsMining":       false,
			"replacesChain":      false,
			"rejectsBlocks":      false,
			"rejectsTx":          false,
		},
	}
}

func stringFromMap(m map[string]any, key string, fallback string) string {
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

func intFromMap(m map[string]any, key string, fallback int) int {
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
