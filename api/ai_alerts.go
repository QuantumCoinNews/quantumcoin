// Package api contains QuantumCoin project code for this module.
package api

import (
	"net/http"

	"quantumcoin/ai"
	"quantumcoin/internal/httpresponse"
)

// HandleAIAlerts documents the QuantumCoin component used by this module.
// GET /api/ai/alerts
func HandleAIAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	alerts := ai.GetAlertsSnapshot()
	httpresponse.WriteJSON(w, http.StatusOK, alerts)
}
