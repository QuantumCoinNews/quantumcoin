package main

import (
	"fmt"
	"net/http"
	"quantumcoin/ai"
	"quantumcoin/p2p"
	"quantumcoin/security"
	"strconv"
	"time"
)

// Security surfaces share one live evidence projection; analysis never enforces fees or transfers.
func requireAPIGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, 405, "method not allowed")
		return false
	}
	return true
}
func handleSecuritySummary(w http.ResponseWriter, r *http.Request) {
	if requireAPIGet(w, r) {
		writeOK(w, security.Summary("."))
	}
}
func handleSecurityScore(w http.ResponseWriter, r *http.Request) {
	if requireAPIGet(w, r) {
		writeOK(w, security.Score("."))
	}
}
func handleSecurityEvents(w http.ResponseWriter, r *http.Request) {
	if !requireAPIGet(w, r) {
		return
	}
	limit := 100
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 {
		limit = n
	}
	if limit > 1000 {
		limit = 1000
	}
	w.Header().Set("X-Page-Limit", strconv.Itoa(limit))
	writeOK(w, security.ReadEvents(".", limit))
}
func livePeerReputation() map[string]any {
	out := nodeRuntime.PeerReputationSnapshot()
	if out["peers"] == nil {
		out["peers"] = map[string]any{}
	}
	if !nodeRuntime.PeerSecurityAvailable() {
		out["overallRisk"] = "UNKNOWN"
		out["status"] = "NO_PEER_SECURITY"
	}
	return out
}
func handlePeerReputation(w http.ResponseWriter, r *http.Request) {
	if requireAPIGet(w, r) {
		writeOK(w, livePeerReputation())
	}
}

type securityView struct {
	dashboard map[string]any
	admin     map[string]any
}

func buildSecurityView(w http.ResponseWriter) securityView {
	sections := nodeRuntime.SecurityRuntimeView()
	bc, _ := nodeRuntime.Snapshot()
	walletObs := make([]security.WalletTxObservation, 0)
	for _, block := range boundedAnalysisBlocksV1(w, bc) {
		if block == nil {
			continue
		}
		for _, tx := range block.Transactions {
			if tx == nil || tx.IsCoinbase() {
				continue
			}
			amount := bc.TransactionAmountQCAtHeightV1(tx.Amount, block.Index)
			walletObs = append(walletObs, security.WalletTxObservation{TxID: fmt.Sprintf("%x", tx.ID), Sender: tx.Sender, Amount: int(amount), AmountQC: amount, Inputs: len(tx.Inputs), Outputs: len(tx.Outputs), BlockIndex: block.Index})
		}
	}
	pending, height := boundedAnalysisPendingV1(w, bc)
	mempoolObs := make([]security.MempoolTxObservation, 0, len(pending))
	for _, tx := range pending {
		if tx == nil {
			continue
		}
		amount := bc.TransactionAmountQCAtHeightV1(tx.Amount, height)
		mempoolObs = append(mempoolObs, security.MempoolTxObservation{TxID: fmt.Sprintf("%x", tx.ID), Sender: tx.Sender, Amount: int(amount), AmountQC: amount, Inputs: len(tx.Inputs), Outputs: len(tx.Outputs)})
	}
	wallet := security.AnalyzeWalletAnomalies(walletObs)
	mempool := security.AnalyzeMempoolAbuse(mempoolObs)
	fees := security.EvaluateDynamicAntiSpamFeePolicy(security.DynamicAntiSpamFeePolicyInput{Observations: mempoolObs, BaseMinFee: 0, MempoolSoftLimit: 25, MempoolHardLimit: 50})
	network := p2p.NetworkViewSnapshot()
	tip := bc.TipSnapshot()
	hash := ""
	age := int64(-1)
	chainHeight := -1
	if tip != nil {
		hash = fmt.Sprintf("%x", tip.Hash)
		age = time.Now().Unix() - tip.Timestamp
		if age < 0 {
			age = 0
		}
		chainHeight = tip.Index
	}
	finality := security.EvaluateFinalityCheckpointWarning(security.FinalityCheckpointInput{ChainHeight: chainHeight, BlockCount: bc.BlockCountSnapshot(), LatestBlockHash: hash, LatestBlockAgeSeconds: age, PeerCount: network.PeerCount, FinalityDepth: 6, ReorgWarningDepth: 3})
	finality["peerHeightKnown"] = network.HeightKnown
	if network.PeerCount > 0 && !network.HeightKnown && finality["status"] == "FINALITY_OK" {
		finality["status"] = "PEER_CONFIRMATION_UNKNOWN"
		finality["risk"] = "WATCH"
	}
	summary, score, peer := security.Summary("."), security.Score("."), livePeerReputation()
	input := security.SecurityAdminReportInput{Summary: summary, Score: score, NodeHealth: sections["nodeHealth"], PeerReputation: peer, MinerSafety: sections["minerSafety"], ActionBurstGuard: sections["actionBurstGuard"], DoubleSpendConflict: sections["doubleSpendConflict"], IPAPIReputation: sections["ipApiReputation"], GUICLIAPIGuardrails: sections["guiCliApiGuardrails"], WalletAnomaly: wallet, MempoolAbuse: mempool, AntiSpamFeePolicy: fees, FinalityCheckpoint: finality}
	admin := security.BuildSecurityAdminReadonlyReport(input)
	adminAlerts := security.BuildAdminSecurityAlerts(security.AdminSecurityAlertInput{AdminReport: admin, EventLog: security.SummarizeEvents(security.ReadEvents(".", 1000).Events)})
	dashboard := map[string]any{"version": "SECURITY_DASHBOARD_API_V1", "summary": summary, "score": score, "nodeHealth": sections["nodeHealth"], "peerReputation": peer, "networkView": network, "walletAnomaly": wallet, "mempoolAbuse": mempool, "antiSpamFeePolicy": fees, "finalityCheckpoint": finality, "actionBurstGuard": sections["actionBurstGuard"], "minerSafety": sections["minerSafety"], "doubleSpendConflict": sections["doubleSpendConflict"], "ipApiReputation": sections["ipApiReputation"], "guiCliApiGuardrails": sections["guiCliApiGuardrails"], "adminAlerts": adminAlerts, "monitoring": nodeRuntime.Monitoring().Snapshot(), "safeBoundary": map[string]bool{"readOnly": true, "changesChain": false, "changesBalances": false, "changesConsensus": false, "changesTotalSupply": false, "performsTransfer": false, "startsMining": false}}
	admin["adminAlerts"] = adminAlerts
	dashboard["endpoints"] = []string{"/api/security/summary", "/api/security/events", "/api/security/score", "/api/security/node-health", "/api/security/peer-reputation", "/api/security/wallet-anomaly", "/api/security/mempool-abuse", "/api/security/anti-spam-fee-policy", "/api/security/finality-checkpoint", "/api/security/dashboard", "/api/security/admin-report", "/api/monitoring"}
	admin["observationScope"] = map[string]any{"http": "last 60 seconds, max 2000 requests; RemoteAddr only", "spendRejections": "last 256 canonical rejections", "miner": "process runtime events", "clientCoverage": "server-observed API; GUI/client internals are not inferred"}
	return securityView{dashboard: dashboard, admin: admin}
}
func handleSecurityDashboard(w http.ResponseWriter, r *http.Request) {
	if requireAPIGet(w, r) {
		writeOK(w, buildSecurityView(w).dashboard)
	}
}
func handleSecurityAdminReport(w http.ResponseWriter, r *http.Request) {
	if requireAPIGet(w, r) {
		writeOK(w, buildSecurityView(w).admin)
	}
}
func handleAISecurityDashboard(w http.ResponseWriter, r *http.Request) {
	if requireAPIGet(w, r) {
		view := buildSecurityView(w)
		view.dashboard["aiBridgeSource"] = "AI_SECURITY_DASHBOARD_API_V1"
		writeOK(w, ai.BuildAISecurityDashboardBridge(ai.AISecurityDashboardBridgeInput{AdminReport: view.admin, Dashboard: view.dashboard}))
	}
}
func securitySection(w http.ResponseWriter, r *http.Request, key string) {
	if requireAPIGet(w, r) {
		writeOK(w, buildSecurityView(w).dashboard[key])
	}
}
func handleWalletAnomaly(w http.ResponseWriter, r *http.Request) {
	securitySection(w, r, "walletAnomaly")
}
func handleMempoolAbuse(w http.ResponseWriter, r *http.Request) {
	securitySection(w, r, "mempoolAbuse")
}
func handleAntiSpamFeePolicy(w http.ResponseWriter, r *http.Request) {
	securitySection(w, r, "antiSpamFeePolicy")
}
func handleFinalityCheckpoint(w http.ResponseWriter, r *http.Request) {
	securitySection(w, r, "finalityCheckpoint")
}
