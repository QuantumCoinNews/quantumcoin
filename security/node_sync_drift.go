// Package security contains QuantumCoin project code for this module.
package security

const NodeSyncDriftDetectorVersion = "NODE_SYNC_DRIFT_DETECTOR_V1"

type NodeSyncDriftInput struct {
	LocalHeight           int   `json:"localHeight"`
	NetworkHeight         int   `json:"networkHeight"`
	MaxPeerHeight         int   `json:"maxPeerHeight"`
	MinPeerHeight         int   `json:"minPeerHeight"`
	PeerCount             int   `json:"peerCount"`
	LatestBlockAgeSeconds int64 `json:"latestBlockAgeSeconds"`
	DriftWarningBlocks    int   `json:"driftWarningBlocks"`
	DriftCriticalBlocks   int   `json:"driftCriticalBlocks"`
}

func EvaluateNodeSyncDrift(input NodeSyncDriftInput) map[string]any {
	warningBlocks := input.DriftWarningBlocks
	if warningBlocks <= 0 {
		warningBlocks = 3
	}

	criticalBlocks := input.DriftCriticalBlocks
	if criticalBlocks <= 0 {
		criticalBlocks = 12
	}

	networkHeight := input.NetworkHeight
	if input.MaxPeerHeight > networkHeight {
		networkHeight = input.MaxPeerHeight
	}
	if networkHeight < 0 {
		networkHeight = 0
	}

	localHeight := input.LocalHeight
	if localHeight < 0 {
		localHeight = 0
	}

	driftBlocks := networkHeight - localHeight
	absoluteDriftBlocks := driftBlocks
	if absoluteDriftBlocks < 0 {
		absoluteDriftBlocks = -absoluteDriftBlocks
	}

	health := "OK"
	syncStatus := "SYNC_OK"
	score := 0
	warnings := []string{}

	if input.PeerCount == 0 {
		health = "ISOLATED"
		syncStatus = "NO_PEERS_LOCAL_ONLY"
		score += 15
		warnings = append(warnings, "node has no connected peers; sync view is local only")
	}

	if networkHeight == 0 && localHeight == 0 {
		if health == "OK" {
			health = "WATCH"
			syncStatus = "NO_HEIGHT_DATA"
		}
		score += 10
		warnings = append(warnings, "no local or network height data available")
	}

	if driftBlocks >= criticalBlocks {
		health = "DEGRADED"
		syncStatus = "NODE_CRITICALLY_BEHIND"
		score += 70
		warnings = append(warnings, "local node height is critically behind network height")
	} else if driftBlocks >= warningBlocks {
		if health == "OK" {
			health = "WATCH"
			syncStatus = "NODE_BEHIND_NETWORK"
		}
		score += 20
		warnings = append(warnings, "local node height is behind network height")
	}

	if networkHeight > 0 && -driftBlocks >= criticalBlocks {
		health = "DEGRADED"
		syncStatus = "LOCAL_CHAIN_AHEAD_OF_NETWORK_VIEW"
		score += 35
		warnings = append(warnings, "local node height is far ahead of network view")
	} else if networkHeight > 0 && -driftBlocks >= warningBlocks {
		if health == "OK" {
			health = "WATCH"
			syncStatus = "LOCAL_CHAIN_AHEAD_OF_NETWORK_VIEW"
		}
		score += 15
		warnings = append(warnings, "local node height is ahead of network view")
	}

	if input.LatestBlockAgeSeconds > 21600 {
		if health == "OK" {
			health = "WATCH"
			syncStatus = "STALE_LOCAL_HEAD"
		}
		score += 35
		warnings = append(warnings, "latest local block is older than 6 hours")
	}

	risk := "LOW"
	switch {
	case score >= 70:
		risk = "HIGH"
	case score >= 35:
		risk = "MEDIUM"
	case score >= 15:
		risk = "WATCH"
	}

	if health == "OK" && risk != "LOW" {
		health = "WATCH"
	}
	if risk == "HIGH" {
		health = "DEGRADED"
	}

	return map[string]any{
		"version":               NodeSyncDriftDetectorVersion,
		"mode":                  "READ_ONLY_NODE_SYNC_DRIFT_ANALYSIS",
		"health":                health,
		"syncStatus":            syncStatus,
		"risk":                  risk,
		"score":                 score,
		"warnings":              warnings,
		"localHeight":           localHeight,
		"networkHeight":         networkHeight,
		"maxPeerHeight":         input.MaxPeerHeight,
		"minPeerHeight":         input.MinPeerHeight,
		"peerCount":             input.PeerCount,
		"latestBlockAgeSeconds": input.LatestBlockAgeSeconds,
		"driftBlocks":           driftBlocks,
		"absoluteDriftBlocks":   absoluteDriftBlocks,
		"driftWarningBlocks":    warningBlocks,
		"driftCriticalBlocks":   criticalBlocks,
		"policyNote":            "This detector only reports node sync drift and health risk for admin/AI/security review. It does not connect peers, replace chain, change consensus, change balances, transfer funds, or start mining.",
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"connectsPeers":      false,
			"disconnectsPeers":   false,
			"replacesChain":      false,
			"changesMempool":     false,
			"changesChain":       false,
			"changesBalances":    false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
			"performsTransfer":   false,
			"startsMining":       false,
		},
	}
}
