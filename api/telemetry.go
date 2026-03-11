package api

import (
	"encoding/json"
	"net/http"

	"quantumcoin/ai"
)

// GET /api/telemetry
func HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	snap := ai.GetTelemetrySnapshot()

	enc := json.NewEncoder(w)
	// Daha okunaklı olsun istersen indent'li:
	// enc.SetIndent("", "  ")

	if err := enc.Encode(snap); err != nil {
		http.Error(w, "failed to encode telemetry", http.StatusInternalServerError)
		return
	}
}
