package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"quantumcoin/ai"
	"quantumcoin/blockchain"
	"quantumcoin/internal"
	qnode "quantumcoin/node"
	"quantumcoin/wallet"
)

func TestConsensusAndAdmissionCannotCallAI(t *testing.T) {
	for _, directory := range []string{"blockchain", "consensus", "internal", "miner", "p2p"} {
		files, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range files {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(directory, name)
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, spec := range file.Imports {
				importPath, _ := strconv.Unquote(spec.Path.Value)
				if importPath != "quantumcoin/ai" {
					continue
				}
				if directory != "blockchain" || (name != "ai_bridge.go" && name != "ai_analysis_sample.go") || spec.Name != nil {
					t.Errorf("AI imported outside advisory projection: %s", path)
					continue
				}
				ast.Inspect(file, func(node ast.Node) bool {
					selection, ok := node.(*ast.SelectorExpr)
					if ok {
						if pkg, ok := selection.X.(*ast.Ident); ok && pkg.Name == "ai" && selection.Sel.Name != "TxLite" {
							t.Errorf("advisory projection uses AI behavior: %s: %s", path, selection.Sel.Name)
						}
					}
					return true
				})
			}
		}
	}
}

func fundAdvisoryOwner(t *testing.T, chain *blockchain.Blockchain, owner *wallet.Wallet, count int) []*blockchain.Transaction {
	t.Helper()
	funds := make([]*blockchain.Transaction, 0, count)
	// Distinct miners also avoid identical legacy coinbases within one second.
	for i := 0; i < count; i++ {
		miner := wallet.NewWallet()
		if _, err := chain.MineBlock(miner.GetAddress(), 8); err != nil {
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
	if _, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 8); err != nil {
		t.Fatal(err)
	}
	return funds
}

func TestLargeSignedTransfersIgnoreAdvisoryState(t *testing.T) {
	wasEnabled := ai.Enabled()
	t.Cleanup(func() { ai.SetEnabled(wasEnabled) })
	for _, enabled := range []bool{false, true} {
		for _, ingress := range []string{"api", "cli", "peer", "adapter", "direct"} {
			t.Run(ingress+map[bool]string{false: "_off", true: "_on"}[enabled], func(t *testing.T) {
				chain, owner, to, c := cliSendFixture(t)
				ai.SetEnabled(enabled)
				fundAdvisoryOwner(t, chain, owner, 10)
				tx, err := blockchain.NewTransactionFromQCAmountV1(owner.GetAddress(), to.GetAddress(), "500", chain)
				if err != nil {
					t.Fatal(err)
				}
				if err = tx.Sign(owner.PrivateKey); err != nil {
					t.Fatal(err)
				}
				switch ingress {
				case "api":
					previous := nodeRuntime
					previousConfig, previousReadOnly := cfg, apiReadOnlyMode
					cfg, apiReadOnlyMode = c, false
					nodeRuntime = qnode.NewRuntime(chain, c)
					defer func() { nodeRuntime, cfg, apiReadOnlyMode = previous, previousConfig, previousReadOnly }()
					body, _ := json.Marshal(SendRequest{Transaction: tx})
					w := httptest.NewRecorder()
					handleSendTx(w, httptest.NewRequest("POST", "/api/tx/send", strings.NewReader(string(body))))
					if w.Code != 200 {
						t.Fatal(w.Code, w.Body.String())
					}
					telemetry := httptest.NewRecorder()
					handleTelemetry(telemetry, httptest.NewRequest("GET", "/api/telemetry", nil))
					var report TelemetryResponse
					if err := json.Unmarshal(telemetry.Body.Bytes(), &report); err != nil || report.MempoolSize != 1 {
						t.Fatalf("telemetry did not read canonical pending transactions: %v, %s", err, telemetry.Body.String())
					}
				case "cli":
					if err = wallet.SaveWallet(owner); err != nil {
						t.Fatal(err)
					}
					tx, err = submitCLISendV1(chain, owner.GetAddress(), to.GetAddress(), "500")
				case "peer":
					err = chain.AddTransactionFromPeerSynchronizedV1(tx)
				case "adapter":
					if !internal.NewCanonicalMempool(chain).Add(tx) {
						t.Fatal("adapter rejected")
					}
				case "direct":
					err = chain.AddTransaction(tx)
				}
				if err != nil {
					t.Fatal(err)
				}
				if chain.PendingTransactionCountSnapshot() != 1 {
					t.Fatal("not admitted")
				}
				if err = chain.AddTransaction(tx); err == nil {
					t.Fatal("canonical duplicate accepted")
				}
			})
		}
	}
}

func TestAdvisorySeesSequentialPendingAndConfirmedBurst(t *testing.T) {
	chain, owner, to, c := cliSendFixture(t)
	funds := fundAdvisoryOwner(t, chain, owner, 5)
	before := ai.Enabled()
	ai.SetEnabled(true)
	defer ai.SetEnabled(before)
	previous := nodeRuntime
	nodeRuntime = qnode.NewRuntime(chain, c)
	defer func() { nodeRuntime = previous }()
	for i := 0; i < 5; i++ {
		tx, err := blockchain.NewTransactionFromQCAmountV1(owner.GetAddress(), to.GetAddress(), "2", chain)
		if err != nil {
			t.Fatal(err)
		}
		// Spend distinct, confirmed 50-QC outputs; pending conflicts remain
		// canonical validation errors rather than part of the AI fixture.
		tx.Inputs = []blockchain.TransactionInput{{TxID: funds[i].ID, OutIndex: 0}}
		tx.ID = tx.Hash()
		if err = tx.Sign(owner.PrivateKey); err != nil {
			t.Fatal(err)
		}
		if err = chain.AddTransaction(tx); err != nil {
			t.Fatal(err)
		}
	}
	check := func() {
		w := httptest.NewRecorder()
		handleAIAnalysis(w, httptest.NewRequest("GET", "/api/ai/analysis", nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "high frequency activity") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	check()
	if chain.PendingTransactionCountSnapshot() != 5 {
		t.Fatal("analysis mutated pool")
	}
	if _, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 8); err != nil {
		t.Fatal(err)
	}
	check()
	if chain.PendingTransactionCountSnapshot() != 0 {
		t.Fatal("analysis recreated pending entries")
	}
}
