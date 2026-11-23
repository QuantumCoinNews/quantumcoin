// Package api contains QuantumCoin project code for this module.
package api

import (
	"net/http"
	"quantumcoin/internal/httpresponse"
)

// AddressProvider is retained for source compatibility but is never called.
// Deprecated: use the canonical node wallet address endpoint.
var AddressProvider func() string

// RegisterWalletRoutes installs only a fail-closed legacy retirement response.
// It does not expose wallet state or provide an alternate node API.
// Deprecated: the canonical root router owns wallet routes and authentication.
func RegisterWalletRoutes(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.HandleFunc("/api/wallet/address", func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusGone
		message := "legacy wallet route retired; use the canonical node API"
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			status = http.StatusMethodNotAllowed
			message = "method not allowed"
		}
		httpresponse.WriteError(w, status, message)
	})
}
