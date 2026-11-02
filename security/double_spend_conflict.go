// Package security contains QuantumCoin project code for this module.
package security

import (
	"sort"
	"strings"
)

const DoubleSpendConflictMonitorVersion = "DOUBLE_SPEND_CONFLICT_MONITOR_V1"

type DoubleSpendConflictObservation struct {
	Subject            string `json:"subject"`
	TxID               string `json:"txid"`
	Outpoint           string `json:"outpoint"`
	Source             string `json:"source"`
	Reason             string `json:"reason"`
	PendingConflict    bool   `json:"pendingConflict"`
	InsideTxConflict   bool   `json:"insideTxConflict"`
	MissingUTXO        bool   `json:"missingUtxo"`
	SpentOrEmptyUTXO   bool   `json:"spentOrEmptyUtxo"`
	InvalidSignature   bool   `json:"invalidSignature"`
	PeerSubmitted      bool   `json:"peerSubmitted"`
	TelegramGamePath   bool   `json:"telegramGamePath"`
	BlockchainCorePath bool   `json:"blockchainCorePath"`
}

type DoubleSpendConflictFinding struct {
	Subject  string         `json:"subject"`
	Risk     string         `json:"risk"`
	Score    int            `json:"score"`
	Reasons  []string       `json:"reasons"`
	Counters map[string]int `json:"counters"`
}

type doubleSpendConflictCounter struct {
	subject            string
	eventCount         int
	pendingConflict    int
	insideTxConflict   int
	missingUTXO        int
	spentOrEmptyUTXO   int
	invalidSignature   int
	peerSubmitted      int
	telegramGamePath   int
	blockchainCorePath int
	outpointSet        map[string]bool
	txIDSet            map[string]bool
}

func AnalyzeDoubleSpendConflicts(observations []DoubleSpendConflictObservation) map[string]any {
	counters := map[string]*doubleSpendConflictCounter{}

	for _, obs := range observations {
		subject := strings.TrimSpace(obs.Subject)
		if subject == "" {
			subject = "UNKNOWN_CONFLICT_SUBJECT"
		}

		c, ok := counters[subject]
		if !ok {
			c = &doubleSpendConflictCounter{
				subject:     subject,
				outpointSet: map[string]bool{},
				txIDSet:     map[string]bool{},
			}
			counters[subject] = c
		}

		c.eventCount++

		if outpoint := strings.TrimSpace(obs.Outpoint); outpoint != "" {
			c.outpointSet[outpoint] = true
		}

		if txid := strings.TrimSpace(obs.TxID); txid != "" {
			c.txIDSet[txid] = true
		}

		reason := strings.ToLower(strings.TrimSpace(obs.Reason))
		source := strings.ToLower(strings.TrimSpace(obs.Source))

		pending := obs.PendingConflict || strings.Contains(reason, "input already pending") || strings.Contains(reason, "pending in mempool")
		inside := obs.InsideTxConflict || strings.Contains(reason, "double spend inside transaction")
		missing := obs.MissingUTXO || strings.Contains(reason, "missing utxo")
		spent := obs.SpentOrEmptyUTXO || strings.Contains(reason, "spent or empty utxo")
		invalidSig := obs.InvalidSignature || strings.Contains(reason, "invalid signature")

		if pending {
			c.pendingConflict++
		}
		if inside {
			c.insideTxConflict++
		}
		if missing {
			c.missingUTXO++
		}
		if spent {
			c.spentOrEmptyUTXO++
		}
		if invalidSig {
			c.invalidSignature++
		}
		if obs.PeerSubmitted || strings.Contains(source, "p2p") || strings.Contains(source, "peer") {
			c.peerSubmitted++
		}
		if obs.TelegramGamePath || strings.Contains(source, "telegram") || strings.Contains(source, "game") {
			c.telegramGamePath++
		}
		if obs.BlockchainCorePath || strings.Contains(source, "blockchain") || strings.Contains(source, "mempool") {
			c.blockchainCorePath++
		}
	}

	findings := make([]DoubleSpendConflictFinding, 0)

	for subject, c := range counters {
		score := 0
		reasons := []string{}

		if c.eventCount >= 20 {
			score += 30
			reasons = append(reasons, "high conflict event count")
		} else if c.eventCount >= 5 {
			score += 15
			reasons = append(reasons, "elevated conflict event count")
		}

		if c.pendingConflict > 0 {
			score += 25 * c.pendingConflict
			reasons = append(reasons, "pending mempool input conflict observed")
		}

		if c.insideTxConflict > 0 {
			score += 30 * c.insideTxConflict
			reasons = append(reasons, "double spend inside transaction observed")
		}

		if c.missingUTXO > 0 {
			score += 12 * c.missingUTXO
			reasons = append(reasons, "missing UTXO conflict observed")
		}

		if c.spentOrEmptyUTXO > 0 {
			score += 20 * c.spentOrEmptyUTXO
			reasons = append(reasons, "spent or empty UTXO conflict observed")
		}

		if c.invalidSignature > 0 {
			score += 8 * c.invalidSignature
			reasons = append(reasons, "invalid signature near conflict path observed")
		}

		if len(c.outpointSet) >= 3 {
			score += 10
			reasons = append(reasons, "multiple outpoints involved in conflicts")
		}

		if c.peerSubmitted > 0 {
			score += 5 * c.peerSubmitted
			reasons = append(reasons, "peer submitted conflict path observed")
		}

		if c.telegramGamePath > 0 {
			score += 5
			reasons = append(reasons, "telegram game conflict path observed")
		}

		if c.blockchainCorePath > 0 {
			score += 5
			reasons = append(reasons, "blockchain core conflict path observed")
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

		if score > 0 {
			findings = append(findings, DoubleSpendConflictFinding{
				Subject: subject,
				Risk:    risk,
				Score:   score,
				Reasons: reasons,
				Counters: map[string]int{
					"eventCount":         c.eventCount,
					"uniqueOutpoints":    len(c.outpointSet),
					"uniqueTxIDs":        len(c.txIDSet),
					"pendingConflict":    c.pendingConflict,
					"insideTxConflict":   c.insideTxConflict,
					"missingUTXO":        c.missingUTXO,
					"spentOrEmptyUTXO":   c.spentOrEmptyUTXO,
					"invalidSignature":   c.invalidSignature,
					"peerSubmitted":      c.peerSubmitted,
					"telegramGamePath":   c.telegramGamePath,
					"blockchainCorePath": c.blockchainCorePath,
				},
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Score == findings[j].Score {
			return findings[i].Subject < findings[j].Subject
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
		"version":              DoubleSpendConflictMonitorVersion,
		"mode":                 "READ_ONLY_DOUBLE_SPEND_CONFLICT_ANALYSIS",
		"overallRisk":          overallRisk,
		"observationsAnalyzed": len(observations),
		"findings":             findings,
		"rules": []string{
			"pending mempool input conflict",
			"double spend inside transaction",
			"missing UTXO conflict",
			"spent or empty UTXO conflict",
			"invalid signature near conflict path",
			"peer submitted conflict path",
			"telegram game conflict path",
			"blockchain core conflict path",
		},
		"policyNote": "This monitor only reports double-spend/conflict risk for admin/AI/security review. It does not reject transactions, mutate mempool, write chain, replace chain, change balances, or perform transfers.",
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"rejectsTx":          false,
			"changesMempool":     false,
			"changesChain":       false,
			"replacesChain":      false,
			"changesBalances":    false,
			"changesConsensus":   false,
			"changesTotalSupply": false,
			"performsTransfer":   false,
			"startsMining":       false,
		},
	}
}
