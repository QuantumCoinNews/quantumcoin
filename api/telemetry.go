// Package api contains QuantumCoin project code for this module.
package api

import (
	"net/http"

	"quantumcoin/ai"
	"quantumcoin/internal/httpresponse"
)

// HandleTelemetry documents the QuantumCoin component used by this module.
// GET /api/telemetry
func HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	snap := ai.GetTelemetrySnapshot()
	httpresponse.WriteJSON(w, http.StatusOK, snap)
}
