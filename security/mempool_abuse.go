// Package security contains QuantumCoin project code for this module.
package security

import (
	"sort"
	"strings"
)

const MempoolAbuseControlVersion = "MEMPOOL_ABUSE_CONTROL_V1"

type MempoolTxObservation struct {
	TxID     string  `json:"txid"`
	Sender   string  `json:"sender"`
	Amount   int     `json:"amount"`
	AmountQC float64 `json:"amountQC,omitempty"`
	Inputs   int     `json:"inputs"`
	Outputs  int     `json:"outputs"`
}

type MempoolAbuseFinding struct {
	Subject       string         `json:"subject"`
	Risk          string         `json:"risk"`
	Score         int            `json:"score"`
	Reasons       []string       `json:"reasons"`
	Counters      map[string]int `json:"counters"`
	TotalAmountQC float64        `json:"totalAmountQC"`
	MaxAmountQC   float64        `json:"maxAmountQC"`
}

type mempoolAbuseCounter struct {
	subject       string
	txCount       int
	totalAmount   int
	maxAmount     int
	totalAmountQC float64
	maxAmountQC   float64
	largeTx       int
	manyInputs    int
	manyOutputs   int
	emptySender   int
	duplicateTxID int
}

func AnalyzeMempoolAbuse(observations []MempoolTxObservation) map[string]any {
	counters := map[string]*mempoolAbuseCounter{}
	txSeen := map[string]bool{}

	for _, obs := range observations {
		subject := strings.TrimSpace(obs.Sender)
		if subject == "" {
			subject = "UNKNOWN_SENDER"
		}

		c, ok := counters[subject]
		if !ok {
			c = &mempoolAbuseCounter{subject: subject}
			counters[subject] = c
		}

		c.txCount++

		amountQC := observationAmountQCV1(obs.Amount, obs.AmountQC)
		if amountQC > 0 {
			c.totalAmount += int(amountQC)
			if int(amountQC) > c.maxAmount {
				c.maxAmount = int(amountQC)
			}
			c.totalAmountQC += amountQC
			if amountQC > c.maxAmountQC {
				c.maxAmountQC = amountQC
			}
		}

		if amountQC >= 1000 {
			c.largeTx++
		}

		if obs.Inputs >= 10 {
			c.manyInputs++
		}

		if obs.Outputs >= 10 {
			c.manyOutputs++
		}

		if strings.TrimSpace(obs.Sender) == "" {
			c.emptySender++
		}

		txid := strings.TrimSpace(obs.TxID)
		if txid != "" {
			if txSeen[txid] {
				c.duplicateTxID++
			}
			txSeen[txid] = true
		}
	}

	findings := make([]MempoolAbuseFinding, 0)

	for subject, c := range counters {
		score := 0
		reasons := []string{}

		if c.txCount >= 25 {
			score += 35
			reasons = append(reasons, "high pending transaction count from same sender")
		} else if c.txCount >= 10 {
			score += 18
			reasons = append(reasons, "elevated pending transaction count from same sender")
		}

		if c.largeTx > 0 {
			score += 10 * c.largeTx
			reasons = append(reasons, "large pending transaction observed")
		}

		if c.manyInputs > 0 {
			score += 8 * c.manyInputs
			reasons = append(reasons, "many-input pending transaction observed")
		}

		if c.manyOutputs > 0 {
			score += 8 * c.manyOutputs
			reasons = append(reasons, "many-output pending transaction observed")
		}

		if c.emptySender > 0 {
			score += 10 * c.emptySender
			reasons = append(reasons, "pending transaction with empty sender observed")
		}

		if c.duplicateTxID > 0 {
			score += 30 * c.duplicateTxID
			reasons = append(reasons, "duplicate pending transaction id observed")
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
			findings = append(findings, MempoolAbuseFinding{
				Subject: subject,
				Risk:    risk,
				Score:   score,
				Reasons: reasons,
				Counters: map[string]int{
					"txCount":       c.txCount,
					"totalAmount":   c.totalAmount,
					"maxAmount":     c.maxAmount,
					"largeTx":       c.largeTx,
					"manyInputs":    c.manyInputs,
					"manyOutputs":   c.manyOutputs,
					"emptySender":   c.emptySender,
					"duplicateTxID": c.duplicateTxID,
				},
				TotalAmountQC: c.totalAmountQC,
				MaxAmountQC:   c.maxAmountQC,
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
		"version":              MempoolAbuseControlVersion,
		"overallRisk":          overallRisk,
		"observationsAnalyzed": len(observations),
		"findings":             findings,
		"rules": []string{
			"high pending tx count per sender",
			"large pending transaction",
			"many-input pending transaction",
			"many-output pending transaction",
			"empty sender pending transaction",
			"duplicate pending tx id",
		},
		"safeBoundary": map[string]bool{
			"readOnly":           true,
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
