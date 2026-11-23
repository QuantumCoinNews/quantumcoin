package node

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"quantumcoin/ai"
	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

func advisoryRuntime(t *testing.T) *Runtime {
	t.Helper()
	cfg := defaultRuntimePolicyConfigV1(t)
	r, err := BootstrapWithConfig(cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	before := ai.Enabled()
	ai.SetEnabled(true)
	t.Cleanup(func() { ai.SetEnabled(before) })
	if r.ObserverCount() != 1 {
		t.Fatal("canonical observer not installed")
	}
	if ai.GetTelemetrySnapshot().Timestamp == 0 {
		t.Fatal("runtime telemetry missing")
	}
	if ai.GetTelemetrySnapshot().HashrateAvailable {
		t.Fatal("invented hashrate")
	}
	return r
}

func TestAdvisoryBootstrapEventShutdownAndReadOnlyPersistence(t *testing.T) {
	r := advisoryRuntime(t)
	if _, err := os.Stat(r.advisoryDir); !os.IsNotExist(err) {
		t.Fatal("startup wrote advisory state")
	}
	r.SetAPIReadOnly(true)
	r.EmitEvent(NodeEvent{Type: "READ_ONLY_TEST", Message: "observed"})
	if _, err := os.Stat(r.advisoryDir); !os.IsNotExist(err) {
		t.Fatal("read-only event wrote disk")
	}
	r.SetAPIReadOnly(false)
	r.EmitEvent(NodeEvent{Type: "AUTH_FAILURE", Severity: NodeEventSeverityWarning, Message: "test event"})
	data, err := os.ReadFile(filepath.Join(r.advisoryDir, "ai_state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("AUTH_FAILURE")) || bytes.Contains(data, []byte(`"bonus"`)) {
		t.Fatal("runtime state incomplete")
	}
	if err = r.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(r.advisoryDir, "ai_state.json"))
	if err != nil || !bytes.Contains(data, []byte("NODE_SHUTDOWN_COMPLETED")) {
		t.Fatal("shutdown state not saved", err)
	}
	// Fresh bootstrap reads the same canonical sidecar, independent of cwd/env.
	// Clear process observations first so an unwired loader cannot pass merely
	// because the earlier runtime's alert is still in memory.
	blankDir := t.TempDir()
	if err = os.WriteFile(filepath.Join(blankDir, "ai_state.json"), []byte(`{"telemetry":{},"alerts":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = ai.LoadStateFrom(blankDir); err != nil {
		t.Fatal(err)
	}
	if len(ai.GetAlertsSnapshot()) != 0 {
		t.Fatal("test failed to clear live observations")
	}
	r2, err := BootstrapWithConfig(r.Config, true)
	if err != nil {
		t.Fatal(err)
	}
	if r2.advisoryDir != r.advisoryDir {
		t.Fatal("advisory path drift")
	}
	found := false
	for _, alert := range ai.GetAlertsSnapshot() {
		if alert.Code == "AUTH_FAILURE" {
			found = true
		}
	}
	if !found {
		t.Fatal("restart lost alert history")
	}
}

func TestAdvisoryDiskFailureCannotChangeMiningSuccess(t *testing.T) {
	r := advisoryRuntime(t)
	if err := os.WriteFile(r.advisoryDir, []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	chain, _ := r.Snapshot()
	beforeHeight := chain.HeightSnapshot()
	beforeFailures := ai.GetPersistenceStatus().Failures
	difficulty, err := chain.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	block, err := r.MineOnceV1(wallet.NewWallet().GetAddress(), difficulty)
	if err != nil || block == nil || chain.HeightSnapshot() != beforeHeight+1 {
		t.Fatal("advisory I/O affected consensus", err)
	}
	if ai.GetPersistenceStatus().Failures <= beforeFailures {
		t.Fatal("runtime persistence failure invisible")
	}
	data, _ := os.ReadFile(r.advisoryDir)
	if string(data) != "occupied" {
		t.Fatal("existing file overwritten")
	}
	if err := chain.ValidateBlockchainSynchronized(); err != nil {
		t.Fatal("advisory changed chain validity", err)
	}
}

func TestAdvisoryMiningReviewReachesCanonicalAlertsAndHistory(t *testing.T) {
	r := advisoryRuntime(t)
	chain, _ := r.Snapshot()
	chain.SetCoinbaseMaturity(0)
	owner, to := wallet.NewWallet(), wallet.NewWallet()
	funds := make([]*blockchain.Transaction, 0, 5)
	for i := 0; i < 5; i++ {
		miner := wallet.NewWallet()
		difficulty, err := chain.ExpectedNextDifficultyV1()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.MineOnceV1(miner.GetAddress(), difficulty); err != nil {
			t.Fatal(err)
		}
		tx, err := blockchain.NewTransactionFromQCAmountV1(miner.GetAddress(), owner.GetAddress(), "50", chain)
		if err != nil {
			t.Fatal(err)
		}
		if err = tx.Sign(miner.PrivateKey); err != nil {
			t.Fatal(err)
		}
		if err = chain.AddTransaction(tx); err != nil {
			t.Fatal(err)
		}
		funds = append(funds, tx)
	}
	mine := func() {
		difficulty, err := chain.ExpectedNextDifficultyV1()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.MineOnceV1(wallet.NewWallet().GetAddress(), difficulty); err != nil {
			t.Fatal(err)
		}
	}
	mine()
	for _, fund := range funds {
		tx, err := blockchain.NewTransactionFromQCAmountV1(owner.GetAddress(), to.GetAddress(), "2", chain)
		if err != nil {
			t.Fatal(err)
		}
		tx.Inputs = []blockchain.TransactionInput{{TxID: fund.ID, OutIndex: 0}}
		tx.ID = tx.Hash()
		if err = tx.Sign(owner.PrivateKey); err != nil {
			t.Fatal(err)
		}
		if err = chain.AddTransaction(tx); err != nil {
			t.Fatal(err)
		}
	}
	mine()
	data, err := os.ReadFile(filepath.Join(r.advisoryDir, "ai_security_reviews.json"))
	if err != nil {
		t.Fatal(err)
	}
	var history ai.AISecurityReviewFile
	if err = json.Unmarshal(data, &history); err != nil || len(history.Reviews) == 0 {
		t.Fatal("missing runtime review", err)
	}
	found := false
	for _, alert := range ai.GetAlertsSnapshot() {
		if alert.Code == "TRANSACTION_REVIEW" {
			found = true
		}
	}
	if !found {
		t.Fatal("review not visible in canonical alert buffer")
	}
	if chain.PendingTransactionCountSnapshot() != 0 {
		t.Fatal("advisory changed pending pool")
	}
}
