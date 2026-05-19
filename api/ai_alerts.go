package api

import (
	"encoding/json"
	"net/http"

	"quantumcoin/ai"
)

// GET /api/ai/alerts
func HandleAIAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	alerts := ai.GetAlertsSnapshot()

	enc := json.NewEncoder(w)
	// Daha okunaklı olsun istersen:
	// enc.SetIndent("", "  ")

	if err := enc.Encode(alerts); err != nil {
		http.Error(w, "failed to encode alerts", http.StatusInternalServerError)
		return
	}
}
