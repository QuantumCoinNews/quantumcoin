// Package security contains QuantumCoin project code for this module.
package security

import "strings"

const DynamicAntiSpamFeePolicyVersion = "DYNAMIC_ANTI_SPAM_FEE_POLICY_V1"

type DynamicAntiSpamFeePolicyInput struct {
	Observations     []MempoolTxObservation `json:"observations"`
	BaseMinFee       int                    `json:"baseMinFee"`
	MempoolSoftLimit int                    `json:"mempoolSoftLimit"`
	MempoolHardLimit int                    `json:"mempoolHardLimit"`
}

func EvaluateDynamicAntiSpamFeePolicy(input DynamicAntiSpamFeePolicyInput) map[string]any {
	baseMinFee := input.BaseMinFee
	if baseMinFee < 0 {
		baseMinFee = 0
	}

	softLimit := input.MempoolSoftLimit
	if softLimit <= 0 {
		softLimit = 25
	}

	hardLimit := input.MempoolHardLimit
	if hardLimit <= softLimit {
		hardLimit = softLimit * 2
	}

	totalPending := len(input.Observations)
	perSender := map[string]int{}
	duplicateTxIDs := 0
	seenTxIDs := map[string]bool{}
	largeTxCount := 0
	manyInputCount := 0
	manyOutputCount := 0
	emptySenderCount := 0

	for _, obs := range input.Observations {
		sender := strings.TrimSpace(obs.Sender)
		if sender == "" {
			sender = "UNKNOWN_SENDER"
		}
		perSender[sender]++

		txid := strings.TrimSpace(obs.TxID)
		if txid != "" {
			if seenTxIDs[txid] {
				duplicateTxIDs++
			}
			seenTxIDs[txid] = true
		}

		if observationAmountQCV1(obs.Amount, obs.AmountQC) >= 1000 {
			largeTxCount++
		}
		if obs.Inputs >= 10 {
			manyInputCount++
		}
		if obs.Outputs >= 10 {
			manyOutputCount++
		}
		if strings.TrimSpace(obs.Sender) == "" {
			emptySenderCount++
		}
	}

	maxSenderPending := 0
	topSender := ""
	for sender, count := range perSender {
		if count > maxSenderPending {
			maxSenderPending = count
			topSender = sender
		}
	}

	pressure := "LOW"
	pressureScore := 0

	if totalPending >= hardLimit {
		pressure = "HIGH"
		pressureScore += 50
	} else if totalPending >= softLimit {
		pressure = "MEDIUM"
		pressureScore += 25
	}

	if maxSenderPending >= 25 {
		pressureScore += 35
	} else if maxSenderPending >= 10 {
		pressureScore += 18
	}

	if duplicateTxIDs > 0 {
		pressureScore += duplicateTxIDs * 30
	}
	if emptySenderCount > 0 {
		pressureScore += emptySenderCount * 10
	}
	if largeTxCount > 0 {
		pressureScore += largeTxCount * 5
	}
	if manyInputCount > 0 {
		pressureScore += manyInputCount * 5
	}
	if manyOutputCount > 0 {
		pressureScore += manyOutputCount * 5
	}

	risk := "LOW"
	switch {
	case pressureScore >= 70:
		risk = "HIGH"
	case pressureScore >= 35:
		risk = "MEDIUM"
	case pressureScore >= 15:
		risk = "WATCH"
	}

	suggestedMinFee := baseMinFee
	switch risk {
	case "WATCH":
		suggestedMinFee = baseMinFee + 1
	case "MEDIUM":
		suggestedMinFee = baseMinFee + 2
	case "HIGH":
		suggestedMinFee = baseMinFee + 5
	}

	return map[string]any{
		"version":         DynamicAntiSpamFeePolicyVersion,
		"mode":            "READ_ONLY_POLICY",
		"risk":            risk,
		"pressure":        pressure,
		"pressureScore":   pressureScore,
		"baseMinFee":      baseMinFee,
		"suggestedMinFee": suggestedMinFee,
		"mempool": map[string]any{
			"totalPending":     totalPending,
			"softLimit":        softLimit,
			"hardLimit":        hardLimit,
			"senderCount":      len(perSender),
			"topSender":        topSender,
			"maxSenderPending": maxSenderPending,
		},
		"signals": map[string]int{
			"duplicateTxIDs":   duplicateTxIDs,
			"largeTxCount":     largeTxCount,
			"manyInputCount":   manyInputCount,
			"manyOutputCount":  manyOutputCount,
			"emptySenderCount": emptySenderCount,
		},
		"policyNote": "This endpoint only recommends an anti-spam fee policy. It does not enforce fees or reject transactions.",
		"safeBoundary": map[string]bool{
			"readOnly":           true,
			"enforcesFee":        false,
			"rejectsTx":          false,
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
