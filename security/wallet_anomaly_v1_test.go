package security

import "testing"

func TestWalletAnomalyDetectorV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := AnalyzeWalletAnomalies(nil)

	if out["version"] != WalletAnomalyDetectorVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], WalletAnomalyDetectorVersion)
	}
	if out["mode"] != "READ_ONLY_WALLET_ANOMALY_ANALYSIS" {
		t.Fatalf("mode mismatch: got=%v", out["mode"])
	}
	if out["overallRisk"] != "LOW" {
		t.Fatalf("empty observations should be LOW, got=%v", out["overallRisk"])
	}
	if out["observationsAnalyzed"] != 0 {
		t.Fatalf("empty observations count mismatch: got=%v", out["observationsAnalyzed"])
	}

	note, ok := out["policyNote"].(string)
	if !ok || note == "" {
		t.Fatalf("policyNote must exist")
	}

	safe, ok := out["safeBoundary"].(map[string]bool)
	if !ok {
		t.Fatalf("safeBoundary type mismatch: %T", out["safeBoundary"])
	}

	if !safe["readOnly"] {
		t.Fatalf("safeBoundary.readOnly must be true")
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

	for _, k := range expectedFalse {
		if safe[k] {
			t.Fatalf("safeBoundary.%s must be false", k)
		}
	}
}

func TestWalletAnomalyDetectorV1DetectsHighRiskWalletPattern(t *testing.T) {
	observations := make([]WalletTxObservation, 0, 50)

	for i := 0; i < 50; i++ {
		observations = append(observations, WalletTxObservation{
			TxID:       "wallet-risk-tx",
			Sender:     "qc_wallet_risk_test",
			Amount:     1500,
			Inputs:     10,
			Outputs:    10,
			BlockIndex: i,
		})
	}

	out := AnalyzeWalletAnomalies(observations)

	if out["version"] != WalletAnomalyDetectorVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], WalletAnomalyDetectorVersion)
	}
	if out["overallRisk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["overallRisk"], out)
	}
	if out["observationsAnalyzed"] != 50 {
		t.Fatalf("observations count mismatch: got=%v", out["observationsAnalyzed"])
	}

	findings, ok := out["findings"].([]WalletAnomalyFinding)
	if !ok {
		t.Fatalf("findings type mismatch: %T", out["findings"])
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding")
	}

	top := findings[0]
	if top.Wallet != "qc_wallet_risk_test" {
		t.Fatalf("top finding wallet mismatch: got=%s", top.Wallet)
	}
	if top.Risk != "HIGH" {
		t.Fatalf("top finding risk mismatch: got=%s score=%d reasons=%v", top.Risk, top.Score, top.Reasons)
	}
	if top.Score < 70 {
		t.Fatalf("top finding score should be >=70, got=%d", top.Score)
	}
	if top.Counters["txCount"] != 50 {
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

func TestWalletAnomalyDetectorV1CoinbaseGroupDoesNotBecomeUserRisk(t *testing.T) {
	observations := []WalletTxObservation{
		{
			TxID:       "coinbase-test",
			Sender:     "COINBASE",
			Amount:     50000,
			Inputs:     0,
			Outputs:    1,
			BlockIndex: 1,
		},
	}

	out := AnalyzeWalletAnomalies(observations)

	if out["overallRisk"] != "LOW" {
		t.Fatalf("coinbase/system group should remain LOW, got=%v out=%v", out["overallRisk"], out)
	}

	findings, ok := out["findings"].([]WalletAnomalyFinding)
	if !ok {
		t.Fatalf("findings type mismatch: %T", out["findings"])
	}
	if len(findings) != 1 {
		t.Fatalf("expected one coinbase/system finding, got=%d", len(findings))
	}
	if findings[0].Score != 0 {
		t.Fatalf("coinbase/system finding score should be 0, got=%d", findings[0].Score)
	}
	if findings[0].Risk != "LOW" {
		t.Fatalf("coinbase/system finding risk should be LOW, got=%s", findings[0].Risk)
	}
}
