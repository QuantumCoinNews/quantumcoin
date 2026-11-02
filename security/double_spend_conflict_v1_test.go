package security

import (
	"os"
	"strings"
	"testing"
)

func TestDoubleSpendConflictMonitorV1LowRiskReadOnlyBoundary(t *testing.T) {
	out := AnalyzeDoubleSpendConflicts(nil)

	if out["version"] != DoubleSpendConflictMonitorVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], DoubleSpendConflictMonitorVersion)
	}
	if out["mode"] != "READ_ONLY_DOUBLE_SPEND_CONFLICT_ANALYSIS" {
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
		"rejectsTx",
		"changesMempool",
		"changesChain",
		"replacesChain",
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

func TestDoubleSpendConflictMonitorV1DetectsHighRiskConflictPattern(t *testing.T) {
	observations := []DoubleSpendConflictObservation{
		{
			Subject:            "wallet_conflict_test",
			TxID:               "tx-a",
			Outpoint:           "prevtx:0",
			Source:             "blockchain_mempool",
			Reason:             "input already pending in mempool: prevtx:0",
			PendingConflict:    true,
			BlockchainCorePath: true,
		},
		{
			Subject:            "wallet_conflict_test",
			TxID:               "tx-b",
			Outpoint:           "prevtx:0",
			Source:             "blockchain_mempool",
			Reason:             "double spend inside transaction: prevtx:0",
			InsideTxConflict:   true,
			BlockchainCorePath: true,
		},
		{
			Subject:          "wallet_conflict_test",
			TxID:             "tx-c",
			Outpoint:         "prevtx:1",
			Source:           "p2p",
			Reason:           "spent or empty utxo: prevtx:1",
			SpentOrEmptyUTXO: true,
			PeerSubmitted:    true,
		},
		{
			Subject:          "wallet_conflict_test",
			TxID:             "tx-d",
			Outpoint:         "prevtx:2",
			Source:           "telegram_game",
			Reason:           "missing utxo: prevtx:2",
			MissingUTXO:      true,
			TelegramGamePath: true,
		},
	}

	out := AnalyzeDoubleSpendConflicts(observations)

	if out["version"] != DoubleSpendConflictMonitorVersion {
		t.Fatalf("version mismatch: got=%v want=%s", out["version"], DoubleSpendConflictMonitorVersion)
	}
	if out["overallRisk"] != "HIGH" {
		t.Fatalf("expected HIGH risk, got=%v out=%v", out["overallRisk"], out)
	}
	if out["observationsAnalyzed"] != 4 {
		t.Fatalf("observations count mismatch: got=%v", out["observationsAnalyzed"])
	}

	findings, ok := out["findings"].([]DoubleSpendConflictFinding)
	if !ok {
		t.Fatalf("findings type mismatch: %T", out["findings"])
	}
	if len(findings) == 0 {
		t.Fatalf("expected at least one finding")
	}

	top := findings[0]
	if top.Subject != "wallet_conflict_test" {
		t.Fatalf("top finding subject mismatch: got=%s", top.Subject)
	}
	if top.Risk != "HIGH" {
		t.Fatalf("top finding risk mismatch: got=%s score=%d reasons=%v", top.Risk, top.Score, top.Reasons)
	}
	if top.Score < 70 {
		t.Fatalf("top finding score should be >=70, got=%d", top.Score)
	}
	if top.Counters["pendingConflict"] == 0 {
		t.Fatalf("pendingConflict counter should be >0")
	}
	if top.Counters["insideTxConflict"] == 0 {
		t.Fatalf("insideTxConflict counter should be >0")
	}
	if top.Counters["spentOrEmptyUTXO"] == 0 {
		t.Fatalf("spentOrEmptyUTXO counter should be >0")
	}
	if top.Counters["missingUTXO"] == 0 {
		t.Fatalf("missingUTXO counter should be >0")
	}
	if top.Counters["peerSubmitted"] == 0 {
		t.Fatalf("peerSubmitted counter should be >0")
	}
	if top.Counters["telegramGamePath"] == 0 {
		t.Fatalf("telegramGamePath counter should be >0")
	}
	if top.Counters["blockchainCorePath"] == 0 {
		t.Fatalf("blockchainCorePath counter should be >0")
	}

	safe := out["safeBoundary"].(map[string]bool)
	if !safe["readOnly"] ||
		safe["rejectsTx"] ||
		safe["changesMempool"] ||
		safe["changesChain"] ||
		safe["replacesChain"] ||
		safe["changesBalances"] ||
		safe["changesConsensus"] ||
		safe["changesTotalSupply"] ||
		safe["performsTransfer"] ||
		safe["startsMining"] {
		t.Fatalf("safeBoundary violated: %+v", safe)
	}
}

func TestDoubleSpendConflictMonitorV1AdminReportSourceVisibility(t *testing.T) {
	data, err := os.ReadFile("admin_report.go")
	if err != nil {
		t.Fatalf("read admin_report.go: %v", err)
	}

	text := string(data)

	required := []string{
		"DoubleSpendConflict",
		"doubleSpendRisk",
		"doubleSpendConflict",
		"Review double-spend/conflict monitor output",
		"changesMempool",
		"changesChain",
		"changesBalances",
		"changesConsensus",
		"changesTotalSupply",
		"performsTransfer",
		"startsMining",
		"replacesChain",
	}

	for _, marker := range required {
		if !strings.Contains(text, marker) {
			t.Fatalf("admin_report.go missing marker %q", marker)
		}
	}
}
