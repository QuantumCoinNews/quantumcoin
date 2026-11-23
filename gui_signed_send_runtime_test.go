package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	qnode "quantumcoin/node"
	"quantumcoin/wallet"
	"quantumcoin/walletsigner"
)

// The child runs the production CLI dispatcher/bootstrap and local signer.
// The wrapper replaces only the test executable's default flag dispatcher.
func TestMain(m *testing.M) {
	if os.Getenv("QC_GUI_REAL_SIGNER_CHILD_V1") == "1" {
		runNodeCLI()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestGUIRealLocalSignerToCanonicalHTTPAdmission(t *testing.T) {
	chain, owner, to, c := networkSigningFixture(t)
	c.DefaultDifficultyBits = config.QCTestnetDefaultDifficultyBitsV1
	t.Setenv("QC_DEFAULT_DIFFICULTY_BITS", "8")
	t.Setenv("QC_NET_MAGIC", config.QCTestnetMagicV1)
	if err := config.ValidateRuntimeNetworkProfileV1(c); err != nil {
		t.Fatal(err)
	}
	chain.SetCoinbaseMaturity(c.CoinbaseMaturity)
	for range c.CoinbaseMaturity {
		if _, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 8); err != nil {
			t.Fatal(err)
		}
	}
	if err := chain.SaveToFile(c.ChainFile); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(filepath.Dir(c.ChainFile), "gui-signing-config.json")
	encoded, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(configPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("QC_GUI_REAL_SIGNER_CHILD_V1", "1")
	t.Setenv("QC_CONFIG_FILE", configPath)
	t.Setenv("QC_DATA_DIR", filepath.Dir(c.ChainFile))
	t.Setenv("QC_CHAIN_FILE", c.ChainFile)
	t.Setenv("QC_HTTP_PORT", c.HTTPPort)
	t.Setenv("QC_P2P_PORT", c.P2PPort)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	previousRuntime, previousConfig, previousMode := nodeRuntime, cfg, apiReadOnlyMode
	runtime, err := qnode.Bootstrap(qnode.BootstrapOptions{ConfigPath: configPath, DataDir: filepath.Dir(c.ChainFile), StrictChain: true})
	if err != nil {
		t.Fatal("canonical parent node bootstrap failed", err)
	}
	chain, c = runtime.Snapshot()
	nodeRuntime = runtime
	cfg = c
	apiReadOnlyMode = false
	t.Cleanup(func() { nodeRuntime, cfg, apiReadOnlyMode = previousRuntime, previousConfig, previousMode })
	before, err := os.ReadFile(c.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	newSubmissionServer := func() *httptest.Server {
		handler := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			if bytes.Contains(raw, []byte(owner.ExportPrivateKeyHex())) {
				t.Error("private key reached HTTP")
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))
			handler.ServeHTTP(w, r)
		}))
	}
	server := newSubmissionServer()
	defer server.Close()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	raw, err := walletsigner.SignLocal(ctx, executable, owner.GetAddress(), to.GetAddress(), "1", owner.ExportPrivateKeyHex(), false)
	if err != nil {
		command := exec.CommandContext(ctx, executable, "sign-send", owner.GetAddress(), to.GetAddress(), "1", "--key-stdin")
		command.Dir = filepath.Dir(executable)
		command.Stdin = strings.NewReader(owner.ExportPrivateKeyHex() + "\n")
		diagnostic, _ := command.CombinedOutput()
		t.Fatal("production local signer failed", err, strings.ReplaceAll(string(diagnostic), owner.ExportPrivateKeyHex(), "[redacted]"))
	}
	after, err := os.ReadFile(c.ChainFile)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("sign-only child changed persisted chain")
	}
	var request SendRequest
	if err = json.Unmarshal(raw, &request); err != nil || request.Transaction == nil || !chain.VerifyTransaction(request.Transaction) {
		t.Fatal("local output is not a valid network-bound signature")
	}
	tx := request.Transaction
	expectedID, err := walletsigner.SignedTransactionIDV1(raw)
	if err != nil {
		t.Fatal(err)
	}
	if expectedID != hex.EncodeToString(tx.ID) {
		t.Fatal("public ID extraction drift")
	}
	for _, mutate := range []struct {
		name   string
		change func(*blockchain.Transaction)
	}{
		{"missing-signature", func(tx *blockchain.Transaction) { tx.Inputs[0].Signature = nil }},
		{"changed-signature", func(tx *blockchain.Transaction) { tx.Inputs[0].Signature[0] ^= 1 }},
		{"recipient", func(tx *blockchain.Transaction) {
			tx.Outputs[0].PubKeyHash = wallet.HashPubKey(wallet.NewWallet().PublicKey)
		}},
		{"amount", func(tx *blockchain.Transaction) { tx.Outputs[0].Amount++; tx.Outputs[1].Amount-- }},
		{"input", func(tx *blockchain.Transaction) { tx.Inputs[0].OutIndex++ }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			// Each adversarial case gets its own real route limiter so rejection
			// proves signature/admission validation rather than a preceding 429.
			caseServer := newSubmissionServer()
			defer caseServer.Close()
			var copyRequest SendRequest
			if err := json.Unmarshal(raw, &copyRequest); err != nil {
				t.Fatal(err)
			}
			mutate.change(copyRequest.Transaction)
			id, err := blockchain.CanonicalTransactionIDV2(copyRequest.Transaction)
			if err != nil {
				t.Fatal(err)
			}
			copyRequest.Transaction.ID = id
			body, err := json.Marshal(copyRequest)
			if err != nil {
				t.Fatal(err)
			}
			_, status, err := walletsigner.Submit(ctx, caseServer.URL+"/api/tx/send", actualMuxTestTokenV1, body)
			if err != nil || status != http.StatusBadRequest || len(chain.PendingTransactionsSnapshot()) != 0 {
				t.Fatalf("modified signed transfer admitted: status %d error %v", status, err)
			}
		})
	}
	// The same spend with a valid signature scoped to mainnet is not testnet authority.
	foreign, err := blockchain.NewNetworkBlockchainV4(c, config.QCNetworkProfileMainnetV1)
	if err != nil {
		t.Fatal(err)
	}
	var foreignRequest SendRequest
	_ = json.Unmarshal(raw, &foreignRequest)
	if err = foreign.SignTransaction(foreignRequest.Transaction, owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	foreignRaw, err := json.Marshal(foreignRequest)
	if err != nil {
		t.Fatal(err)
	}
	if _, status, err := walletsigner.Submit(ctx, server.URL+"/api/tx/send", actualMuxTestTokenV1, foreignRaw); err != nil || status != http.StatusBadRequest {
		t.Fatalf("cross-network rejection: status=%d error=%v", status, err)
	}
	if _, status, err := walletsigner.Submit(ctx, server.URL+"/api/tx/send", "", raw); err != nil || status != http.StatusUnauthorized {
		t.Fatal("missing node token accepted")
	}
	response, status, err := walletsigner.Submit(ctx, server.URL+"/api/tx/send", actualMuxTestTokenV1, raw)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := walletsigner.ValidateSendResponseV1(status, response, expectedID); err != nil || id != expectedID || len(chain.PendingTransactionsSnapshot()) != 1 {
		t.Fatal("valid signed GUI transfer failed admission", err)
	}
	if _, status, err := walletsigner.Submit(ctx, server.URL+"/api/tx/send", actualMuxTestTokenV1, raw); err != nil || status != http.StatusBadRequest || len(chain.PendingTransactionsSnapshot()) != 1 {
		t.Fatal("signed replay was accepted twice")
	}
	if _, err = walletsigner.SignLocal(ctx, executable, owner.GetAddress(), to.GetAddress(), "1", wallet.NewWallet().ExportPrivateKeyHex(), false); err == nil {
		t.Fatal("wrong owner's local key accepted")
	}
}
