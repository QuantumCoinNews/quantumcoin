// Package security contains QuantumCoin project code for this module.
package security

import "sort"

const PeerReputationReviewVersion = "P2P_REPUTATION_REVIEW_V1"

type PeerReputationObservation struct {
	PeerID                string `json:"peerId"`
	Score                 int    `json:"score"`
	Status                string `json:"status"`
	IsBanned              bool   `json:"isBanned"`
	InvalidMagicCount     int    `json:"invalidMagicCount"`
	InvalidChainCount     int    `json:"invalidChainCount"`
	RejectedTxCount       int    `json:"rejectedTxCount"`
	FloodOrOversizeCount  int    `json:"floodOrOversizeCount"`
	ConnectionRejectCount int    `json:"connectionRejectCount"`
	UnknownMessageCount   int    `json:"unknownMessageCount"`
	HandshakeFailureCount int    `json:"handshakeFailureCount"`
}

type PeerReputationFinding struct {
	PeerID   string         `json:"peerId"`
	Risk     string         `json:"risk"`
	Score    int            `json:"score"`
	Reasons  []string       `json:"reasons"`
	Counters map[string]int `json:"counters"`
}

func AnalyzePeerReputation(observations []PeerReputationObservation) map[string]any {
	findings := make([]PeerReputationFinding, 0)

	for _, obs := range observations {
		peerID := obs.PeerID
		if peerID == "" {
			peerID = "UNKNOWN_PEER"
		}

		riskScore := 0
		reasons := []string{}

		if obs.IsBanned {
			riskScore += 40
			reasons = append(reasons, "peer is currently banned")
		}

		if obs.Score < 20 {
			riskScore += 60
			reasons = append(reasons, "peer reputation score is critically low")
		} else if obs.Score < 50 {
			riskScore += 30
			reasons = append(reasons, "peer reputation score is low")
		} else if obs.Score < 80 {
			riskScore += 10
			reasons = append(reasons, "peer reputation score requires watch")
		}

		if obs.InvalidMagicCount > 0 {
			riskScore += obs.InvalidMagicCount * 25
			reasons = append(reasons, "invalid network magic observed")
		}

		if obs.InvalidChainCount > 0 {
			riskScore += obs.InvalidChainCount * 20
			reasons = append(reasons, "invalid peer chain observed")
		}

		if obs.RejectedTxCount > 0 {
			riskScore += obs.RejectedTxCount * 8
			reasons = append(reasons, "peer transaction rejected by validation/mempool")
		}

		if obs.FloodOrOversizeCount > 0 {
			riskScore += obs.FloodOrOversizeCount * 30
			reasons = append(reasons, "message flood or oversize peer behavior observed")
		}

		if obs.ConnectionRejectCount > 0 {
			riskScore += obs.ConnectionRejectCount * 10
			reasons = append(reasons, "peer connection rejection observed")
		}

		if obs.UnknownMessageCount > 0 {
			riskScore += obs.UnknownMessageCount * 6
			reasons = append(reasons, "unknown peer message type observed")
		}

		if obs.HandshakeFailureCount > 0 {
			riskScore += obs.HandshakeFailureCount * 12
			reasons = append(reasons, "peer handshake failure observed")
		}

		risk := "LOW"
		switch {
		case riskScore >= 70:
			risk = "HIGH"
		case riskScore >= 35:
			risk = "MEDIUM"
		case riskScore >= 15:
			risk = "WATCH"
		}

		if riskScore > 0 {
			findings = append(findings, PeerReputationFinding{
				PeerID:  peerID,
				Risk:    risk,
				Score:   riskScore,
				Reasons: reasons,
				Counters: map[string]int{
					"reputationScore":       obs.Score,
					"invalidMagicCount":     obs.InvalidMagicCount,
					"invalidChainCount":     obs.InvalidChainCount,
					"rejectedTxCount":       obs.RejectedTxCount,
					"floodOrOversizeCount":  obs.FloodOrOversizeCount,
					"connectionRejectCount": obs.ConnectionRejectCount,
					"unknownMessageCount":   obs.UnknownMessageCount,
					"handshakeFailureCount": obs.HandshakeFailureCount,
				},
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Score == findings[j].Score {
			return findings[i].PeerID < findings[j].PeerID
		}
		return findings[i].Score > findings[j].Score
	})

	overallRisk := "LOW"
	if len(findings) > 0 {
		switch {
		case findings[0].Score >= 70:
			overallRisk = "HIGH"
		case findings[0].Score >= 35:
			overallRisk = "MEDIUM"
		case findings[0].Score >= 15:
			overallRisk = "WATCH"
		}
	}

	return map[string]any{
		"version":              PeerReputationReviewVersion,
		"mode":                 "READ_ONLY_P2P_REPUTATION_REVIEW",
		"overallRisk":          overallRisk,
		"observationsAnalyzed": len(observations),
		"findings":             findings,
		"policyNote":           "This review only reports P2P peer reputation and ban-scoring risk for admin/AI/security review. It does not ban peers, connect peers, disconnect peers, replace chain, change consensus, change balances, transfer funds, or start mining.",
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"bansPeer":           false,
			"adjustsReputation":  false,
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
