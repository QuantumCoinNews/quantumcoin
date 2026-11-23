package main

import (
	"net/http"
	"strings"
)

// Read-only mode is a capability boundary, not a method-only filter: some
// legacy GET routes change scores, generate wallets or allocate mining work.
// New API routes are denied until explicitly reviewed as read operations.
func apiReadOnlyRequestAllowedV1(r *http.Request) bool {
	path := r.URL.Path
	if path != "/api" && !strings.HasPrefix(path, "/api/") {
		return r.Method == http.MethodGet || r.Method == http.MethodHead
	}
	// Signed receipt lookup is a read expressed as POST because the proof is
	// carried in a JSON body. It cannot mint, mine or persist chain changes.
	if path == "/api/nft/monthly/receipt" {
		return r.Method == http.MethodPost
	}
	if r.Method != http.MethodGet {
		return false
	}
	switch path {
	case "/api/monitoring", "/api/capabilities", "/api/health", "/api/network/label", "/api/telemetry",
		"/api/economy/policy", "/api/nft/monthly/program",
		"/api/security/summary", "/api/security/events", "/api/security/score",
		"/api/security/node-health", "/api/security/peer-reputation",
		"/api/security/dashboard", "/api/security/admin-report",
		"/api/security/wallet-anomaly", "/api/security/mempool-abuse",
		"/api/security/anti-spam-fee-policy", "/api/security/finality-checkpoint",
		"/api/wallet/address", "/api/tx/status", "/api/ai/alerts",
		"/api/ai/bonus", "/api/ai/analysis", "/api/ai/security-dashboard",
		"/api/blocks", "/api/block", "/api/mine/tip", "/api/mine/result",
		"/api/stake/status", "/api/miner/status":
		return true
	default:
		return strings.HasPrefix(path, "/api/wallet/balance/")
	}
}

func withAPIReadOnlyBoundaryV1(next http.Handler) http.Handler {
	// Capture the configured mode before listening. A reload or a different
	// test/application instance must not silently downgrade this server.
	runtime := nodeRuntime
	readOnly := apiReadOnlyMode || runtime != nil && runtime.IsAPIReadOnly()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		active := readOnly || runtime != nil && runtime.IsAPIReadOnly()
		if active && !apiReadOnlyRequestAllowedV1(r) {
			// Keep header authentication mandatory even for denied capabilities.
			if !checkAPIToken(w, r) {
				return
			}
			writeError(w, http.StatusForbidden, "API mode is read-only")
			return
		}
		next.ServeHTTP(w, r)
	})
}
