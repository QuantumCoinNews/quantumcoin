// Package security contains QuantumCoin project code for this module.
package security

import (
	"sort"
	"strings"
)

const WalletAnomalyDetectorVersion = "WALLET_ANOMALY_DETECTOR_V1"

type WalletTxObservation struct {
	TxID       string  `json:"txid"`
	Sender     string  `json:"sender"`
	Amount     int     `json:"amount"`
	AmountQC   float64 `json:"amountQC,omitempty"`
	Inputs     int     `json:"inputs"`
	Outputs    int     `json:"outputs"`
	BlockIndex int     `json:"blockIndex"`
}

type WalletAnomalyFinding struct {
	Wallet        string         `json:"wallet"`
	Risk          string         `json:"risk"`
	Score         int            `json:"score"`
	Reasons       []string       `json:"reasons"`
	Counters      map[string]int `json:"counters"`
	TotalAmountQC float64        `json:"totalAmountQC"`
	MaxAmountQC   float64        `json:"maxAmountQC"`
}

type walletAnomalyCounter struct {
	wallet        string
	txCount       int
	totalAmount   int
	maxAmount     int
	totalAmountQC float64
	maxAmountQC   float64
	zeroOrNoSend  int
	largeTx       int
	manyInputs    int
	manyOutputs   int
}

func AnalyzeWalletAnomalies(observations []WalletTxObservation) map[string]any {
	counters := map[string]*walletAnomalyCounter{}

	for _, obs := range observations {
		wallet := strings.TrimSpace(obs.Sender)
		if wallet == "" {
			wallet = "UNKNOWN_OR_COINBASE"
		}

		c, ok := counters[wallet]
		if !ok {
			c = &walletAnomalyCounter{wallet: wallet}
			counters[wallet] = c
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

		if wallet == "UNKNOWN_OR_COINBASE" || strings.EqualFold(wallet, "COINBASE") {
			c.zeroOrNoSend++
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
	}

	findings := make([]WalletAnomalyFinding, 0)

	for wallet, c := range counters {
		if wallet == "" {
			continue
		}

		score := 0
		reasons := []string{}

		if c.txCount >= 50 {
			score += 25
			reasons = append(reasons, "high transaction frequency")
		} else if c.txCount >= 20 {
			score += 12
			reasons = append(reasons, "elevated transaction frequency")
		}

		if c.largeTx > 0 {
			score += 15 * c.largeTx
			reasons = append(reasons, "large transaction amount observed")
		}

		if c.manyInputs > 0 {
			score += 10 * c.manyInputs
			reasons = append(reasons, "many-input transaction observed")
		}

		if c.manyOutputs > 0 {
			score += 10 * c.manyOutputs
			reasons = append(reasons, "many-output transaction observed")
		}

		if strings.EqualFold(wallet, "UNKNOWN_OR_COINBASE") || strings.EqualFold(wallet, "COINBASE") {
			score = 0
			reasons = []string{"system or coinbase transaction group"}
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

		if score > 0 || strings.EqualFold(wallet, "UNKNOWN_OR_COINBASE") || strings.EqualFold(wallet, "COINBASE") {
			findings = append(findings, WalletAnomalyFinding{
				Wallet:  wallet,
				Risk:    risk,
				Score:   score,
				Reasons: reasons,
				Counters: map[string]int{
					"txCount":      c.txCount,
					"totalAmount":  c.totalAmount,
					"maxAmount":    c.maxAmount,
					"largeTx":      c.largeTx,
					"manyInputs":   c.manyInputs,
					"manyOutputs":  c.manyOutputs,
					"zeroOrNoSend": c.zeroOrNoSend,
				},
				TotalAmountQC: c.totalAmountQC,
				MaxAmountQC:   c.maxAmountQC,
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Score == findings[j].Score {
			return findings[i].Wallet < findings[j].Wallet
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
		"version":              WalletAnomalyDetectorVersion,
		"mode":                 "READ_ONLY_WALLET_ANOMALY_ANALYSIS",
		"overallRisk":          overallRisk,
		"observationsAnalyzed": len(observations),
		"findings":             findings,
		"rules": []string{
			"high transaction frequency",
			"large transaction amount",
			"many-input transaction",
			"many-output transaction",
			"coinbase/system wallet group is not treated as user anomaly",
		},
		"policyNote": "This analyzer only reports wallet anomaly risk for admin/AI/security review. It does not change balances, chain, mempool, consensus, supply, or perform transfers.",
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
