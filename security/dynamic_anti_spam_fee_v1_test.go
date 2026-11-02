package security

import "testing"

func TestDynamicAntiSpamFeePolicyV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := EvaluateDynamicAntiSpamFeePolicy(DynamicAntiSpamFeePolicyInput{
		Observations:     nil,
		BaseMinFee:       -5,
		MempoolSoftLimit: 0,
		MempoolHardLimit: 0,
	})

	if out["version"] != DynamicAntiSpamFeePolicyVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], DynamicAntiSpamFeePolicyVersion)
	}
	if out["mode"] != "READ_ONLY_POLICY" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["risk"] != "LOW" {
		t.Fatalf("empty observations should be LOW, got=%v", out["risk"])
	}
	if out["pressure"] != "LOW" {
		t.Fatalf("empty observations pressure should be LOW, got=%v", out["pressure"])
	}
	if out["baseMinFee"] != 0 {
		t.Fatalf("negative base fee must normalize to 0, got=%v", out["baseMinFee"])
	}
	if out["suggestedMinFee"] != 0 {
		t.Fatalf("LOW risk suggested fee must stay at base fee, got=%v", out["suggestedMinFee"])
	}

	mempool, ok := out["mempool"].(map[string]any)
	if !ok {
		t.Fatalf("mempool type mismatch: %T", out["mempool"])
	}
	if mempool["totalPending"] != 0 {
		t.Fatalf("totalPending mismatch: got=%v", mempool["totalPending"])
	}
	if mempool["softLimit"] != 25 {
		t.Fatalf("default softLimit mismatch: got=%v", mempool["softLimit"])
	}
	if mempool["hardLimit"] != 50 {
		t.Fatalf("default hardLimit mismatch: got=%v", mempool["hardLimit"])
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}

	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
	}

	expectedFalse := []string{
		"enforcesFee",
		"rejectsTx",
		"changesMempool",
		"changesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"startsMining",
	}

	for _, k := range expectedFalse {
		if safe[k] {
			t.Fatalf("safeBoundary.%s must be false", k)
		}
	}
}

func TestDynamicAntiSpamFeePolicyV1HighRiskRecommendationOnly(t *testing.T) {
	observations := make([]MempoolTxObservation, 0, 50)

	for i := 0; i < 50; i++ {
		txid := "unique-tx"
		if i >= 10 {
			txid = "duplicate-tx-id"
		}

		observations = append(observations, MempoolTxObservation{
			TxID:    txid,
			Sender:  "wallet_fee_pressure_test",
			Amount:  1500,
			Inputs:  10,
			Outputs: 10,
		})
	}

	out := EvaluateDynamicAntiSpamFeePolicy(DynamicAntiSpamFeePolicyInput{
		Observations:     observations,
		BaseMinFee:       1,
		MempoolSoftLimit: 10,
		MempoolHardLimit: 20,
	})

	if out["version"] != DynamicAntiSpamFeePolicyVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], DynamicAntiSpamFeePolicyVersion)
	}
	if out["mode"] != "READ_ONLY_POLICY" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["risk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["risk"], out)
	}
	if out["pressure"] != "HIGH" {
		t.Fatalf("expected HIGH pressure, got=%v", out["pressure"])
	}
	if out["baseMinFee"] != 1 {
		t.Fatalf("baseMinFee mismatch: got=%v", out["baseMinFee"])
	}
	if out["suggestedMinFee"] != 6 {
		t.Fatalf("HIGH risk should recommend base+5 fee only, got=%v", out["suggestedMinFee"])
	}

	mempool, ok := out["mempool"].(map[string]any)
	if !ok {
		t.Fatalf("mempool type mismatch: %T", out["mempool"])
	}
	if mempool["totalPending"] != 50 {
		t.Fatalf("totalPending mismatch: got=%v", mempool["totalPending"])
	}
	if mempool["maxSenderPending"] != 50 {
		t.Fatalf("maxSenderPending mismatch: got=%v", mempool["maxSenderPending"])
	}
	if mempool["topSender"] != "wallet_fee_pressure_test" {
		t.Fatalf("topSender mismatch: got=%v", mempool["topSender"])
	}

	signals, ok := out["signals"].(map[string]int)
	if !ok {
		t.Fatalf("signals type mismatch: %T", out["signals"])
	}
	if signals["duplicateTxIDs"] == 0 {
		t.Fatalf("duplicateTxIDs should be >0")
	}
	if signals["largeTxCount"] == 0 {
		t.Fatalf("largeTxCount should be >0")
	}
	if signals["manyInputCount"] == 0 {
		t.Fatalf("manyInputCount should be >0")
	}
	if signals["manyOutputCount"] == 0 {
		t.Fatalf("manyOutputCount should be >0")
	}

	note, ok := out["policyNote"].(string)
	if !ok || note == "" {
		t.Fatalf("policyNote must exist")
	}

	safe := out["safeBoundary"].(map[string]bool)
	if !safe["readOnly"] ||
		safe["enforcesFee"] ||
		safe["rejectsTx"] ||
		safe["changesMempool"] ||
		safe["changesChain"] ||
		safe["changesBalances"] ||
		safe["changesConsensus"] ||
		safe["changesTotalSupply"] ||
		safe["performsTransfer"] ||
		safe["startsMining"] {
		t.Fatalf("safeBoundary violated: %+v", safe)
	}
}
