package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	qnode "quantumcoin/node"
	"quantumcoin/webui"
	"strings"
	"time"
)

// ServeMux canonicalizes dot segments and duplicate slashes with an HTML
// redirect before dispatch. Reject API namespace aliases before that redirect
// so an API request cannot turn into successful SPA navigation.
func withAPICanonicalPathBoundaryV1(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := path.Clean(r.URL.Path)
		inAPI := r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") ||
			clean == "/api" || strings.HasPrefix(clean, "/api/")
		// Keep legitimate subtree trailing slashes; ServeMux preserves them too.
		if strings.HasSuffix(r.URL.Path, "/") && clean != "/" {
			clean += "/"
		}
		if inAPI && clean != r.URL.Path {
			writeError(w, http.StatusNotFound, "api endpoint not found")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func registerAPINotFoundBoundary(mux *http.ServeMux) {
	if mux == nil {
		return
	}

	notFound := func(w http.ResponseWriter, _ *http.Request) {
		writeError(
			w,
			http.StatusNotFound,
			"api endpoint not found",
		)
	}

	mux.HandleFunc("/api", notFound)
	mux.HandleFunc("/api/", notFound)
}

func newHTTPAPIHandlerV1(
	guard *HTTPGuard,
	crit *EndpointLimiter,
) http.Handler {
	return newHTTPAPIHandlerForListenV1(guard, crit, getAPIListenAddr())
}

func newHTTPAPIHandlerForListenV1(
	guard *HTTPGuard,
	crit *EndpointLimiter,
	listenAddr string,
) http.Handler {
	mux := http.NewServeMux()

	if crit == nil {
		crit = NewEndpointLimiter()
	}

	// Temel API uçları
	mux.HandleFunc("/api/monitoring", protected(crit.WrapFunc("monitoring_reads", 10*time.Second, 10, withAnalysisBudgetV1(handleMonitoring))))
	mux.HandleFunc("/api/security/events/action", crit.WrapFunc("security_event_actions", 10*time.Second, 10, protected(handleMonitoringAction)))
	mux.HandleFunc("/api/monitoring/action", crit.WrapFunc("monitoring_actions", 10*time.Second, 10, protected(handleMonitoringAction)))
	mux.HandleFunc("/api/monitoring/publisher", crit.WrapFunc("publisher_monitor", 10*time.Second, 10, handlePublisherMonitoring))
	mux.HandleFunc("/api/health", protected(handleHealth))
	mux.HandleFunc("/api/capabilities", protected(handleCapabilities))
	mux.HandleFunc("/api/network/label", handleNetworkLabelV1)
	mux.HandleFunc("/api/telemetry", protected(handleTelemetry))
	mux.HandleFunc("/api/economy/policy", protected(handleEconomyPolicy))
	mux.HandleFunc("/api/runtime/policy", protected(crit.WrapFunc("runtime_policy", 10*time.Second, 12, handleRuntimePolicy)))
	mux.HandleFunc("/api/nft/monthly/program", protected(func(w http.ResponseWriter, r *http.Request) {
		if nodeRuntime == nil {
			writeError(w, http.StatusServiceUnavailable, "node runtime unavailable")
			return
		}
		nodeRuntime.ServeNFTGameProgramV1(w, r)
	}))
	mux.HandleFunc(
		"/api/nft/monthly/mint",
		crit.WrapFunc("nft_monthly_mint", 10*time.Second, 5, protected(func(w http.ResponseWriter, r *http.Request) {
			if nodeRuntime == nil {
				writeError(w, http.StatusServiceUnavailable, "node runtime unavailable")
				return
			}
			nodeRuntime.ServeNFTMonthlyMintV1(w, r)
		})),
	)
	mux.HandleFunc(
		"/api/nft/monthly/receipt",
		crit.WrapFunc("nft_monthly_receipt", 10*time.Second, 10, protected(func(w http.ResponseWriter, r *http.Request) {
			if nodeRuntime == nil {
				writeError(w, http.StatusServiceUnavailable, "node runtime unavailable")
				return
			}
			nodeRuntime.ServeNFTMonthlyReceiptV1(w, r)
		})),
	)
	mux.HandleFunc("/api/security/summary", protected(handleSecuritySummary))
	mux.HandleFunc("/api/security/events", protected(crit.WrapFunc("security_reads", 10*time.Second, 10, withAnalysisBudgetV1(handleSecurityEvents))))
	mux.HandleFunc("/api/security/score", protected(handleSecurityScore))
	mux.HandleFunc("/api/security/node-health", protected(handleNodeHealth))
	mux.HandleFunc("/api/security/peer-reputation", protected(handlePeerReputation))
	mux.HandleFunc("/api/security/dashboard", protected(crit.WrapFunc("bounded_analysis", 10*time.Second, 4, withAnalysisBudgetV1(handleSecurityDashboard))))
	mux.HandleFunc("/api/security/admin-report", protected(crit.WrapFunc("bounded_analysis", 10*time.Second, 4, withAnalysisBudgetV1(handleSecurityAdminReport))))
	mux.HandleFunc("/api/security/wallet-anomaly", protected(crit.WrapFunc("bounded_analysis", 10*time.Second, 4, withAnalysisBudgetV1(handleWalletAnomaly))))
	mux.HandleFunc("/api/security/mempool-abuse", protected(crit.WrapFunc("bounded_analysis", 10*time.Second, 4, withAnalysisBudgetV1(handleMempoolAbuse))))
	mux.HandleFunc("/api/security/anti-spam-fee-policy", protected(crit.WrapFunc("bounded_analysis", 10*time.Second, 4, withAnalysisBudgetV1(handleAntiSpamFeePolicy))))
	mux.HandleFunc("/api/security/finality-checkpoint", protected(handleFinalityCheckpoint))
	mux.HandleFunc("/api/wallet/new", protected(handleNewWallet))
	mux.HandleFunc("/api/wallet/address", protected(handleWalletAddress))
	mux.HandleFunc("/api/wallet/balance/", protected(handleBalance))
	mux.HandleFunc("/api/mine", protected(handleMineBlock))
	mux.HandleFunc("/api/mine/local-release", protected(handleIsolatedLocalMineBlock))

	// Kritik uç: tx/send – 10 saniyede en fazla 5 istek.
	mux.HandleFunc(
		"/api/tx/send",
		crit.WrapFunc(
			"tx_send",
			10*time.Second,
			5,
			protected(handleSendTx),
		),
	)
	mux.HandleFunc("/api/tx/status", protected(handleTxStatus))
	mux.HandleFunc("/api/dev/fastmine", protected(handleFastMine))

	// Advisory AI reads.
	mux.HandleFunc("/api/ai/alerts", protected(handleAIAlerts))
	mux.HandleFunc("/api/ai/bonus", protected(handleAIBonus))
	mux.HandleFunc("/api/ai/analysis", protected(crit.WrapFunc("bounded_analysis", 10*time.Second, 4, withAnalysisBudgetV1(handleAIAnalysis))))
	mux.HandleFunc("/api/ai/security-dashboard", protected(crit.WrapFunc("bounded_analysis", 10*time.Second, 4, withAnalysisBudgetV1(handleAISecurityDashboard))))

	// Explorer.
	mux.HandleFunc("/api/blocks", protected(handleBlocksList))
	mux.HandleFunc("/api/block", protected(handleBlockDetail))

	// Burn ve stake.
	mux.HandleFunc(
		"/api/tx/burn",
		crit.WrapFunc(
			"tx_burn",
			10*time.Second,
			5,
			protected(handleBurn),
		),
	)
	mux.HandleFunc("/api/stake/start", protected(handleStakeStart))
	mux.HandleFunc("/api/stake/status", protected(handleStakeStatus))

	// Web miner.
	mux.HandleFunc("/api/mine/job", protected(handleMineJob))
	mux.HandleFunc("/api/mine/tip", protected(handleMiningTip))
	mux.HandleFunc("/api/mine/result", protected(handleMiningResult))
	mux.HandleFunc(
		"/api/mine/submit",
		crit.WrapFunc(
			"mine_submit",
			10*time.Second,
			20,
			protected(handleMineSubmit),
		),
	)

	// Miner kontrol.
	mux.HandleFunc(
		"/api/miner/start",
		protected(crit.WrapFunc(
			"miner_start",
			10*time.Second,
			3,
			handleMinerStart,
		)),
	)
	mux.HandleFunc(
		"/api/miner/stop",
		protected(crit.WrapFunc(
			"miner_stop",
			10*time.Second,
			3,
			handleMinerStop,
		)),
	)
	mux.HandleFunc("/api/miner/status", protected(handleMinerStatus))

	// Bilinmeyen API yolları hiçbir zaman SPA index'e düşmemeli.
	registerAPINotFoundBoundary(mux)

	// Public node landing and browser mining tool.
	if h, err := webui.Handler(); err == nil {
		mux.Handle("/", h)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			writeJSON(
				w,
				http.StatusOK,
				map[string]any{
					"ok":      true,
					"service": "QuantumCoin API",
				},
			)
		})
	}

	var handler http.Handler = withCORSForListenV1(withAPIReadOnlyBoundaryV1(withAPICanonicalPathBoundaryV1(mux)), listenAddr)
	if guard != nil {
		handler = guard.Wrap(handler)
	}

	return withEarlyRejectionBodyDrainV1(observeRuntimeHTTP(handler))
}

func startHTTPAPI() {
	listenAddr := getAPIListenAddr()
	if err := validateAPIBindAuthV1(listenAddr); err != nil {
		log.Printf("http server startup refused: %v", err)
		return
	}
	handler := newHTTPAPIHandlerForListenV1(
		defaultHTTPGuard,
		criticalLimiter,
		listenAddr,
	)

	fmt.Println("HTTP API starting at http://" + listenAddr)

	if err := nodeRuntime.ServeHTTP(
		listenAddr,
		handler,
		qnode.DefaultHTTPServerOptionsV1(),
	); err != nil {
		log.Printf(
			"http server error: %v",
			err,
		)
	}
}
