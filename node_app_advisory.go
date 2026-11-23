package main

import (
	"net/http"
	"quantumcoin/ai"
	"quantumcoin/internal"
)

//lint:ignore U1000 retained disabled AI bonus stub; AI must not affect consensus or supply
func processAIBonus() {
	// AI bonus is disabled for mining consensus and supply safety.
	// AI can monitor/analyze only; it must not mint, write bonus files, or alter balances.
}

func handleAIBonus(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	address := r.URL.Query().Get("address")
	bonuses := internal.ListBonuses(address)
	writeOK(w, bonuses)
}

func handleAIAnalysis(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	address := r.URL.Query().Get("address")

	bc, _ := nodeRuntime.Snapshot()
	if !ai.Enabled() {
		analysisHeadersV1(w, false)
		writeOK(w, map[string]any{
			"anomaly_report":     []any{},
			"recommendations":    []any{},
			"reward_suggestions": []any{},
		})
		return
	}
	lite, truncated := bc.AIAnalysisSample(address)
	analysisHeadersV1(w, truncated)

	anoms := ai.AnalyzeTransactions(lite)
	recs := ai.BuildWalletRecommendations(anoms)
	rewards := ai.OptimizeRewards(lite, anoms)

	writeOK(w, map[string]any{
		"anomaly_report":     anoms,
		"recommendations":    recs,
		"reward_suggestions": rewards,
	})
}

func handleAIAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	list := ai.GetAlertsSnapshot()
	writeOK(w, list)
}
