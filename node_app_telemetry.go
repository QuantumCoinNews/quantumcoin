package main

import (
	"net/http"
	"quantumcoin/ai"
	"quantumcoin/apiclient"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	"runtime"
	"strings"
	"time"
)

// ECONOMY_POLICY_V1
// GET /api/economy/policy
// Read-only economy policy endpoint.
// Safe boundary: no mint, no transfer, no balance change.
func handleEconomyPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, blockchain.EconomyPolicySummary())
}

/* ---------- Handler fonksiyonları ---------- */

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if !requireAPIGet(w, r) {
		return
	}
	// Health verisi hassas durum bilgisi içerdiği için route düzeyinde korunur.
	bc, _ := nodeRuntime.Snapshot()
	if bc == nil {
		writeError(w, http.StatusServiceUnavailable, "node chain is not ready")
		return
	}
	writeOK(w, map[string]any{
		"ok":              true,
		"service":         apiclient.HealthServiceV1,
		"healthVersion":   apiclient.HealthProtocolV1,
		"capabilities":    config.RuntimeCapabilitiesV1(),
		"networkIdentity": bc.NetworkIdentitySnapshotV1(),
		"apiReadOnly":     apiReadOnlyMode || nodeRuntime.IsAPIReadOnly(),
		"height":          bc.BestHeightSnapshot(),
		"time":            time.Now().UTC().Format(time.RFC3339),
		"httpPort":        getHTTPPort(),
	})
}

// /api/telemetry
func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if !requireAPIGet(w, r) {
		return
	}
	reloadAPIChainFromDiskIfChanged()

	height := -1
	blockCount := 0
	totalSupply := 0
	currentReward := 0
	mempoolSize := 0

	bc, _ := nodeRuntime.Snapshot()
	if bc != nil {
		height = bc.BestHeightSnapshot()
		blockCount = bc.BlockCountSnapshot()
		totalSupply = bc.TotalSupplySnapshot()
		mempoolSize = bc.PendingTransactionCountSnapshot()
		nextHeight := height + 1
		currentReward = bc.ExpectedSubsidyForHeightSnapshot(
			nextHeight,
		)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	nodeRuntime.RefreshAdvisoryTelemetry()

	writeOK(w, TelemetryResponse{
		AI:            ai.GetTelemetrySnapshot(),
		AIPersistence: ai.GetPersistenceStatus(),
		Height:        height,
		BlockCount:    blockCount,
		Peers:         nodeRuntime.PeerCount(),
		MempoolSize:   mempoolSize,
		MinerRunning:  nodeRuntime != nil && nodeRuntime.IsMining(),
		HTTPPort:      getHTTPPort(),
		P2PPort:       strings.TrimPrefix(cfg.P2PPort, ":"),
		ChainFile:     cfg.ChainFile,
		TotalSupply:   totalSupply,
		CurrentReward: currentReward,
		CPUCount:      runtime.NumCPU(),
		GoRoutines:    runtime.NumGoroutine(),
		MemMB:         float64(m.Alloc) / (1024 * 1024),
	})
}

// NODE_HEALTH_SUMMARY_V1
// NODE_SYNC_DRIFT_DETECTOR_V1
// GET /api/security/node-health
func handleNodeHealth(w http.ResponseWriter, r *http.Request) {
	if !requireAPIGet(w, r) {
		return
	}
	reloadAPIChainFromDiskIfChanged()
	result := nodeRuntime.SecurityRuntimeView()["nodeHealth"]
	result["syncVersion"] = result["version"]
	drift, _ := result["driftBlocks"].(int)
	age, _ := result["latestBlockAgeSeconds"].(int64)
	result["driftWarning"] = drift >= 3 || age > 21600
	bc, _ := nodeRuntime.Snapshot()
	result["version"] = "NODE_HEALTH_SUMMARY_V1"
	result["lastBlockAgeSeconds"] = result["latestBlockAgeSeconds"]
	result["mempoolSize"] = bc.PendingTransactionCountSnapshot()
	result["totalSupply"] = bc.TotalSupplySnapshot()
	result["minerRunning"] = nodeRuntime.IsMining()
	result["cpuCount"] = runtime.NumCPU()
	result["goRoutines"] = runtime.NumGoroutine()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	result["memoryMB"] = float64(mem.Alloc) / (1024 * 1024)
	writeOK(w, result)
}
