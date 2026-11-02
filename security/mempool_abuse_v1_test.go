package security

import (
	"fmt"
	"testing"
)

func TestMempoolAbuseControlV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := AnalyzeMempoolAbuse(nil)

	if out["version"] != MempoolAbuseControlVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], MempoolAbuseControlVersion)
	}
	if out["overallRisk"] != "LOW" {
		t.Fatalf("empty observations should be LOW, got=%v", out["overallRisk"])
	}
	if out["observationsAnalyzed"] != 0 {
		t.Fatalf("empty observations count mismatch: got=%v", out["observationsAnalyzed"])
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}

	expectedFalse := []string{
		"changesMempool",
		"changesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"startsMining",
	}

	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
	}

	for _, k := range expectedFalse {
		if safe[k] {
			t.Fatalf("safeBoundary.%s must be false", k)
		}
	}
}

func TestMempoolAbuseControlV1DetectsHighRiskPattern(t *testing.T) {
	observations := make([]MempoolTxObservation, 0, 25)

	for i := 0; i < 25; i++ {
		txid := fmt.Sprintf("tx-%02d", i)
		if i >= 20 {
			txid = "duplicate-tx-id"
		}

		observations = append(observations, MempoolTxObservation{
			TxID:    txid,
			Sender:  "wallet_abuse_test",
			Amount:  1500,
			Inputs:  10,
			Outputs: 10,
		})
	}

	out := AnalyzeMempoolAbuse(observations)

	if out["version"] != MempoolAbuseControlVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], MempoolAbuseControlVersion)
	}
	if out["observationsAnalyzed"] != 25 {
		t.Fatalf("observations count mismatch: got=%v", out["observationsAnalyzed"])
	}
	if out["overallRisk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["overallRisk"], out)
	}

	findings, ok := out["findings"].([]MempoolAbuseFinding)
	if !ok {
		t.Fatalf("findings type mismatch: %T", out["findings"])
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding")
	}

	top := findings[0]
	if top.Subject != "wallet_abuse_test" {
		t.Fatalf("top finding subject mismatch: got=%s", top.Subject)
	}
	if top.Risk != "HIGH" {
		t.Fatalf("top finding risk mismatch: got=%s score=%d reasons=%v", top.Risk, top.Score, top.Reasons)
	}
	if top.Score < 70 {
		t.Fatalf("top finding score should be >=70, got=%d", top.Score)
	}
	if top.Counters["txCount"] != 25 {
		t.Fatalf("txCount mismatch: got=%d", top.Counters["txCount"])
	}
	if top.Counters["largeTx"] == 0 {
		t.Fatalf("largeTx counter should be >0")
	}
	if top.Counters["manyInputs"] == 0 {
		t.Fatalf("manyInputs counter should be >0")
	}
	if top.Counters["manyOutputs"] == 0 {
		t.Fatalf("manyOutputs counter should be >0")
	}
	if top.Counters["duplicateTxID"] == 0 {
		t.Fatalf("duplicateTxID counter should be >0")
	}

	safe := out["safeBoundary"].(map[string]bool)
	if !safe["readOnly"] ||
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
