package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"quantumcoin/blockchain"
	qconsensus "quantumcoin/consensus"
	qnode "quantumcoin/node"
	"quantumcoin/wallet"
	"strings"
	"testing"
)

func signedAPIFixture(t *testing.T) (http.Handler, *blockchain.Blockchain, *wallet.Wallet, *wallet.Wallet) {
	t.Helper()
	chain, owner, to, c := cliSendFixture(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	previousRuntime, previousConfig, previousMode := nodeRuntime, cfg, apiReadOnlyMode
	nodeRuntime = qnode.NewRuntime(chain, c)
	cfg = c
	apiReadOnlyMode = false
	t.Cleanup(func() {
		nodeRuntime, cfg, apiReadOnlyMode = previousRuntime, previousConfig, previousMode
	})
	return newHTTPAPIHandlerV1(nil, NewEndpointLimiter()), chain, owner, to
}
func signedEnvelope(t *testing.T, tx *blockchain.Transaction) string {
	t.Helper()
	raw, err := json.Marshal(SendRequest{Transaction: tx})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestWalletHTTPRejectsRawKeysAndAcceptsOnlyLocallySignedTransactions(t *testing.T) {
	h, chain, owner, to := signedAPIFixture(t)
	tx, err := signWalletIntentV1(chain, owner.GetAddress(), to.GetAddress(), "2", owner.ExportPrivateKeyHex(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.PendingTransactionsSnapshot()) != 0 {
		t.Fatal("local signing submitted transaction")
	}
	if runtimeChain, _ := nodeRuntime.Snapshot(); runtimeChain != chain || !runtimeChain.VerifyTransaction(tx) {
		t.Fatal("runtime chain does not verify local wallet signature before HTTP submission")
	}
	legacy := `{"from":"` + owner.GetAddress() + `","to":"` + to.GetAddress() + `","amount":"2","priv_hex":"[REDACTED_SECRET]"}`
	w := webRequest(h, "POST", "/api/tx/send", legacy)
	if w.Code != 400 || strings.Contains(w.Body.String(), owner.ExportPrivateKeyHex()) || len(chain.PendingTransactionsSnapshot()) != 0 {
		t.Fatal("legacy key body accepted or echoed", w.Code)
	}
	envelope := signedEnvelope(t, tx)
	withSecret := strings.TrimSuffix(envelope, "}") + `,"priv_hex":"[REDACTED_SECRET]"}`
	if w := webRequest(h, "POST", "/api/tx/send", withSecret); w.Code != 400 {
		t.Fatal("unknown key field ignored")
	}
	request := httptest.NewRequest("POST", "/api/tx/send", strings.NewReader(envelope))
	request.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
	request.RemoteAddr = "203.0.113.1:1234"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != 200 || !strings.Contains(w.Body.String(), hex.EncodeToString(tx.ID)) {
		t.Fatal(w.Code, w.Body.String())
	}
	if len(chain.PendingTransactionsSnapshot()) != 1 {
		t.Fatal("signed transaction not admitted")
	}
	if w := webRequest(h, "POST", "/api/tx/send", envelope); w.Code != 400 {
		t.Fatal("duplicate signed transfer accepted")
	}
}

func TestWalletHTTPBurnRequiresActualSignedCanonicalBurn(t *testing.T) {
	h, chain, owner, to := signedAPIFixture(t)
	policy := qconsensus.ActivationPolicyV1{Enabled: true, Plan: qconsensus.ActivationPlan{NetworkID: "signed-burn-api", LegacyGenesisHex: hex.EncodeToString(chain.BlockByIndexSnapshot(0).Hash), ActivationHeight: 2, TargetProtocol: qconsensus.ProtocolAtomV3, PreserveHistory: true, PreserveBalances: true}}
	if err := chain.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatal(err)
	}
	normal, err := signWalletIntentV1(chain, owner.GetAddress(), to.GetAddress(), "0.000001", owner.ExportPrivateKeyHex(), false)
	if err != nil {
		t.Fatal(err)
	}
	if w := webRequest(h, "POST", "/api/tx/burn", signedEnvelope(t, normal)); w.Code != 400 {
		t.Fatal("normal transfer accepted as burn")
	}
	burn, err := signWalletIntentV1(chain, owner.GetAddress(), "", "0.000001", owner.ExportPrivateKeyHex(), true)
	if err != nil {
		t.Fatal(err)
	}
	if w := webRequest(h, "POST", "/api/tx/send", signedEnvelope(t, burn)); w.Code != 400 {
		t.Fatal("burn accepted as normal send")
	}
	unsigned := blockchain.DeserializeBlock((&blockchain.Block{Transactions: []*blockchain.Transaction{burn}}).Serialize()).Transactions[0]
	for i := range unsigned.Inputs {
		unsigned.Inputs[i].Signature = nil
	}
	if w := webRequest(h, "POST", "/api/tx/burn", signedEnvelope(t, unsigned)); w.Code != 400 {
		t.Fatal("unsigned burn accepted")
	}
	if w := webRequest(h, "POST", "/api/tx/burn", signedEnvelope(t, burn)); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if len(chain.PendingTransactionsSnapshot()) != 1 {
		t.Fatal("burn not admitted exactly once")
	}
}

func TestWalletHTTPDoesNotCreateUnbackedKeysAndAdvertisesUnavailableStake(t *testing.T) {
	h, _, _, _ := signedAPIFixture(t)
	before, _ := os.ReadDir(filepath.Dir(cfg.ChainFile))
	for method, status := range map[string]int{"GET": 405, "POST": 410, "PUT": 405} {
		if w := webRequest(h, method, "/api/wallet/new", `{}`); w.Code != status || strings.Contains(w.Body.String(), `"address"`) {
			t.Fatal(method, w.Code, w.Body.String())
		}
	}
	after, _ := os.ReadDir(filepath.Dir(cfg.ChainFile))
	if len(before) != len(after) {
		t.Fatal("wallet endpoint wrote a file")
	}
	for _, endpoint := range []struct{ method, path string }{{"POST", "/api/stake/start"}, {"GET", "/api/stake/status"}} {
		w := webRequest(h, endpoint.method, endpoint.path, `{}`)
		if w.Code != 501 || !strings.Contains(w.Body.String(), `"available":false`) || !strings.Contains(w.Body.String(), `"success":false`) {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w := webRequest(h, "GET", "/api/capabilities", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"staking":false`) || !strings.Contains(w.Body.String(), `"signing":"wallet-local"`) {
		t.Fatal(w.Body.String())
	}
}

func TestWalletAPIRetiredGameCannotCreateAnAlternativeEconomy(t *testing.T) {
	_, _, _, _ = signedAPIFixture(t)
	t.Setenv("QC_GAME_SCORE_PUBLIC_KEY", strings.Repeat("ab", 32))
	h := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	if w := webRequest(h, "GET", "/api/game/score?player=p1&score=999", ""); w.Code != http.StatusNotFound {
		t.Fatal(w.Code)
	}
	for _, path := range []string{"/api/game/score", "/api/game/leaderboard"} {
		req := httptest.NewRequest("GET", path, nil)
		req.RemoteAddr = "127.0.0.1:32100"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatal("removed route is still registered", path, w.Code)
		}
	}
	for _, endpoint := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/game/score", `{"player":"p1","score":999}`},
		{http.MethodPost, "/api/game/score?player=p1&score=-999", `{"player":"p1","score":-999}`},
		{http.MethodGet, "/api/game/leaderboard", ""},
	} {
		w := webRequest(h, endpoint.method, endpoint.path, endpoint.body)
		if w.Code != http.StatusNotFound ||
			!strings.Contains(w.Body.String(), "api endpoint not found") ||
			w.Header().Get("Location") != "" {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(cfg.ChainFile), "game_scores.json")); !os.IsNotExist(err) {
		t.Fatal("retired root created a score store")
	}
}

func TestWalletAPIBoundedAnalysisAndConcurrency(t *testing.T) {
	h, chain, _, _ := signedAPIFixture(t)
	for i := 0; i < blockchain.AnalysisMaxBlocks+2; i++ {
		chain.Blocks = append(chain.Blocks, &blockchain.Block{Index: i + 2, Transactions: []*blockchain.Transaction{}})
	}
	w := webRequest(h, "GET", "/api/ai/analysis", "")
	if w.Code != 200 || w.Header().Get("X-Analysis-Truncated") != "true" {
		t.Fatal(w.Code, w.Header())
	}
	w = webRequest(h, "GET", "/api/security/events?limit=9223372036854775807", "")
	if w.Code != 200 || w.Header().Get("X-Page-Limit") != "1000" {
		t.Fatal(w.Code, w.Header())
	}
	for i := 0; i < 4; i++ {
		w = webRequest(h, "GET", "/api/ai/analysis", "")
	}
	if w.Code != 429 {
		t.Fatal("expensive analysis was not rate limited", w.Code)
	}
	// Check the shared concurrency admission independently of CPU timing.
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	done := make(chan struct{}, 2)
	guarded := withAnalysisBudgetV1(func(w http.ResponseWriter, r *http.Request) { entered <- struct{}{}; <-release; w.WriteHeader(204) })
	for i := 0; i < 2; i++ {
		go func() { guarded(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil)); done <- struct{}{} }()
	}
	<-entered
	<-entered
	rr := httptest.NewRecorder()
	guarded(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != 429 {
		t.Fatal("analysis concurrency unbounded")
	}
	close(release)
	<-done
	<-done
}

func TestWalletLocalSigningCLIOnlyReturnsPublicEnvelope(t *testing.T) {
	chain, owner, to, c := cliSendFixture(t)
	chain.SetCoinbaseMaturity(c.CoinbaseMaturity)
	for i := 0; i < c.CoinbaseMaturity; i++ {
		if _, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 8); err != nil {
			t.Fatal(err)
		}
	}
	if err := chain.SaveToFile(c.ChainFile); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(c.ChainFile)
	exe, _ := os.Executable()
	args, _ := json.Marshal([]string{"sign-send", owner.GetAddress(), to.GetAddress(), "2", "--key-stdin"})
	cmd := exec.Command(exe, "-test.run=^TestCLISendCommandChild$", "-test.v")
	cmd.Dir = filepath.Dir(c.ChainFile)
	cmd.Env = append(os.Environ(), "QC_CLI_SEND_TEST_CHILD=1", "QC_CLI_SEND_TEST_ARGS="+string(args),
		"QC_CHAIN_FILE="+c.ChainFile, "QC_WALLET_FILE="+c.WalletFile, "QC_HTTP_PORT="+c.HTTPPort, "QC_P2P_PORT="+c.P2PPort)
	cmd.Stdin = strings.NewReader(owner.ExportPrivateKeyHex() + "\n")
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal("local signing child failed", err)
	}
	if bytes.Contains(raw, []byte(owner.ExportPrivateKeyHex())) {
		t.Fatal("local signer leaked private key")
	}
	found := false
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "SIGNED_TRANSACTION_V1=") {
			var envelope SendRequest
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "SIGNED_TRANSACTION_V1=")), &envelope) != nil || envelope.Transaction == nil || !chain.VerifyTransaction(envelope.Transaction) {
				t.Fatalf("invalid local signature: txPresent=%t legacyVerify=%t chainVerify=%t identity=%+v", envelope.Transaction != nil, envelope.Transaction != nil && envelope.Transaction.Verify(), envelope.Transaction != nil && chain.VerifyTransaction(envelope.Transaction), chain.NetworkIdentitySnapshotV1())
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing signed envelope")
	}
	after, _ := os.ReadFile(c.ChainFile)
	if !bytes.Equal(before, after) {
		t.Fatal("offline signing wrote chain")
	}
	if _, err := os.Stat(c.WalletFile); !os.IsNotExist(err) {
		t.Fatal("stdin signing created wallet file")
	}
}
