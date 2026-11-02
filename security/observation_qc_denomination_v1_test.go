package security

import "testing"

func TestWalletObservationQCDenominationV1(t *testing.T) {
	oneQC := AnalyzeWalletAnomalies([]WalletTxObservation{{
		Sender: "one-qc", Amount: 1_000_000, AmountQC: 1,
	}})
	if findings := oneQC["findings"].([]WalletAnomalyFinding); len(findings) != 0 {
		t.Fatalf("1 QC was classified as large: %+v", findings)
	}

	micro := AnalyzeWalletAnomalies([]WalletTxObservation{{
		Sender: "one-atom", Amount: 1, AmountQC: 0.000001, Inputs: 10,
	}})
	findings := micro["findings"].([]WalletAnomalyFinding)
	if len(findings) != 1 || findings[0].TotalAmountQC != 0.000001 || findings[0].Counters["largeTx"] != 0 {
		t.Fatalf("one atom lost or misclassified: %+v", findings)
	}
}

func TestMempoolObservationQCDenominationV1(t *testing.T) {
	oneQC := AnalyzeMempoolAbuse([]MempoolTxObservation{{
		Sender: "one-qc", Amount: 1_000_000, AmountQC: 1,
	}})
	if findings := oneQC["findings"].([]MempoolAbuseFinding); len(findings) != 0 {
		t.Fatalf("1 QC was classified as large: %+v", findings)
	}

	micro := AnalyzeMempoolAbuse([]MempoolTxObservation{{
		Sender: "one-atom", Amount: 1, AmountQC: 0.000001, Inputs: 10,
	}})
	findings := micro["findings"].([]MempoolAbuseFinding)
	if len(findings) != 1 || findings[0].TotalAmountQC != 0.000001 || findings[0].Counters["largeTx"] != 0 {
		t.Fatalf("one atom lost or misclassified: %+v", findings)
	}

	fee := EvaluateDynamicAntiSpamFeePolicy(DynamicAntiSpamFeePolicyInput{
		Observations: []MempoolTxObservation{{Sender: "one-qc", Amount: 1_000_000, AmountQC: 1}},
	})
	signals := fee["signals"].(map[string]int)
	if signals["largeTxCount"] != 0 {
		t.Fatalf("1 QC raised large transaction signal: %+v", signals)
	}
}
