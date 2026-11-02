// Package security contains QuantumCoin project code for this module.
package security

const FinalityCheckpointWarningVersion = "FINALITY_CHECKPOINT_WARNING_V1"

type FinalityCheckpointInput struct {
	ChainHeight           int    `json:"chainHeight"`
	BlockCount            int    `json:"blockCount"`
	LatestBlockHash       string `json:"latestBlockHash"`
	LatestBlockAgeSeconds int64  `json:"latestBlockAgeSeconds"`
	PeerCount             int    `json:"peerCount"`
	FinalityDepth         int    `json:"finalityDepth"`
	ReorgWarningDepth     int    `json:"reorgWarningDepth"`
}

func EvaluateFinalityCheckpointWarning(input FinalityCheckpointInput) map[string]any {
	finalityDepth := input.FinalityDepth
	if finalityDepth <= 0 {
		finalityDepth = 6
	}

	reorgWarningDepth := input.ReorgWarningDepth
	if reorgWarningDepth <= 0 {
		reorgWarningDepth = 3
	}

	status := "FINALITY_OK"
	risk := "LOW"
	score := 0
	warnings := []string{}

	if input.BlockCount <= 0 {
		status = "NO_CHAIN_DATA"
		risk = "WATCH"
		score += 15
		warnings = append(warnings, "no chain data available")
	}

	if input.ChainHeight < finalityDepth {
		status = "CHAIN_NOT_DEEP_ENOUGH_FOR_FINALITY"
		if risk == "LOW" {
			risk = "WATCH"
		}
		score += 10
		warnings = append(warnings, "chain height is below configured finality depth")
	}

	if input.PeerCount == 0 {
		if status == "FINALITY_OK" {
			status = "LOCAL_ONLY_FINALITY_VIEW"
		}
		if risk == "LOW" {
			risk = "WATCH"
		}
		score += 10
		warnings = append(warnings, "node has no connected peers; finality view is local only")
	}

	if input.LatestBlockAgeSeconds > 21600 {
		status = "STALE_CHAIN_FINALITY_WARNING"
		risk = "MEDIUM"
		score += 35
		warnings = append(warnings, "latest block is older than 6 hours")
	}

	if input.LatestBlockHash == "" {
		if status == "FINALITY_OK" {
			status = "MISSING_HEAD_HASH"
		}
		if risk == "LOW" {
			risk = "WATCH"
		}
		score += 10
		warnings = append(warnings, "latest block hash is empty")
	}

	if score >= 70 {
		risk = "HIGH"
	} else if score >= 35 {
		risk = "MEDIUM"
	} else if score >= 15 && risk == "LOW" {
		risk = "WATCH"
	}

	confirmedDepth := input.ChainHeight - finalityDepth
	if confirmedDepth < 0 {
		confirmedDepth = 0
	}

	return map[string]any{
		"version":               FinalityCheckpointWarningVersion,
		"mode":                  "READ_ONLY_WARNING",
		"status":                status,
		"risk":                  risk,
		"score":                 score,
		"warnings":              warnings,
		"chainHeight":           input.ChainHeight,
		"blockCount":            input.BlockCount,
		"latestBlockHash":       input.LatestBlockHash,
		"latestBlockAgeSeconds": input.LatestBlockAgeSeconds,
		"peerCount":             input.PeerCount,
		"finalityDepth":         finalityDepth,
		"reorgWarningDepth":     reorgWarningDepth,
		"confirmedDepth":        confirmedDepth,
		"policyNote":            "This endpoint only reports finality and reorg risk warnings. It does not replace chain, reject blocks, or change consensus.",
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"replacesChain":      false,
			"rejectsBlocks":      false,
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
