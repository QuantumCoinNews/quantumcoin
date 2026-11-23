package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/miner"
	"quantumcoin/wallet"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func externalMiningMux(t *testing.T) http.Handler {
	t.Helper()
	readOnlyMuxFixtureV1(t)
	apiReadOnlyMode = false
	nodeRuntime.SetAPIReadOnly(false)
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	webJobs = map[string]*webJob{}
	return newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
}
func webRequest(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = "127.0.0.1:8888"
	r.Host = "127.0.0.1:8082"
	r.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func issueMiningJob(t *testing.T, h http.Handler, addr string) WebMineJobResp {
	t.Helper()
	w := webRequest(h, "GET", "/api/mine/job?address="+addr, "")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var j WebMineJobResp
	if err := json.Unmarshal(w.Body.Bytes(), &j); err != nil {
		t.Fatal(err)
	}
	return j
}
func solveMiningJob(t *testing.T, j WebMineJobResp, start uint64) (uint64, string) {
	t.Helper()
	header, err := hex.DecodeString(j.Header)
	if err != nil {
		t.Fatal(err)
	}
	if j.Protocol != webMiningProtocol || j.NonceOffset != len(header)-8 {
		t.Fatal("invalid header contract")
	}
	for n := start; n < start+1000000; n++ {
		binary.BigEndian.PutUint64(header[j.NonceOffset:], n)
		hash := sha256.Sum256(header)
		if hash[0] == 0 {
			return n, hex.EncodeToString(hash[:])
		}
	}
	t.Fatal("no solution")
	return 0, ""
}
func miningSubmitBody(j WebMineJobResp, n uint64) string {
	b, _ := json.Marshal(map[string]any{"address": j.Miner, "job_id": j.JobID, "nonce": strconv.FormatUint(n, 10)})
	return string(b)
}

func TestWebMiningCommitsClientNonceOnceAndDoesNotInvalidateOtherMiners(t *testing.T) {
	h := externalMiningMux(t)
	chain, _ := nodeRuntime.Snapshot()
	a := issueMiningJob(t, h, wallet.NewWallet().GetAddress())
	b := issueMiningJob(t, h, wallet.NewWallet().GetAddress())
	if a.JobID == b.JobID {
		t.Fatal("shared job ID")
	}
	// Deliberately exceed JS Number's exact integer range; protocol uses strings.
	nonce, hash := solveMiningJob(t, a, 1<<53+100)
	body := miningSubmitBody(a, nonce)
	var wg sync.WaitGroup
	results := make(chan *httptest.ResponseRecorder, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- webRequest(h, "POST", "/api/mine/submit", body) }()
	}
	wg.Wait()
	close(results)
	accepted := 0
	for w := range results {
		if w.Code == 200 {
			var result WebMineSubmitResp
			_ = json.Unmarshal(w.Body.Bytes(), &result)
			if !result.Accepted || result.Hash != hash || result.Height != 1 {
				t.Fatal(w.Body.String())
			}
			accepted++
		} else if w.Code != 409 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if accepted != 1 || chain.BestHeightSnapshot() != 1 {
		t.Fatalf("accepted=%d height=%d", accepted, chain.BestHeightSnapshot())
	}
	tip := chain.TipSnapshot()
	if uint64(tip.Nonce) != nonce || hex.EncodeToString(tip.Hash) != hash || tip.Miner != a.Miner {
		t.Fatal("client block was replaced")
	}
	loaded, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(cfg.ChainFile, chain)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(loaded.TipSnapshot().Hash, tip.Hash) {
		t.Fatal("ack before persistence")
	}
	// Issuing B did not invalidate A. Once A commits, B's old tip is stale.
	bn, _ := solveMiningJob(t, b, 0)
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(b, bn)); w.Code != 409 {
		t.Fatal("stale work accepted", w.Code)
	}
}

func TestStandaloneNodeMinerCommitsCanonicalJobThroughMux(t *testing.T) {
	h := externalMiningMux(t)
	chain, _ := nodeRuntime.Snapshot()
	server := httptest.NewServer(h)
	defer server.Close()
	address := wallet.NewWallet().GetAddress()
	ctx := context.Background()
	unauthorized, err := miner.NewNodeBackend(server.URL, "wrong-token")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unauthorized.GetWork(ctx, address); err == nil {
		t.Fatal("node issued work without valid credentials")
	}
	backend, err := miner.NewNodeBackend(server.URL, actualMuxTestTokenV1)
	if err != nil {
		t.Fatal(err)
	}
	work, err := backend.GetWork(ctx, address)
	if err != nil {
		t.Fatal(err)
	}
	job := WebMineJobResp{Protocol: webMiningProtocol, Header: hex.EncodeToString(work.Header), NonceOffset: len(work.Header) - 8}
	nonce, hash := solveMiningJob(t, job, 1<<53+100)
	if accepted, err := backend.Submit(ctx, work, nonce, hash); err != nil || !accepted {
		t.Fatalf("expected durable acceptance, accepted=%v err=%v", accepted, err)
	}
	if accepted, err := backend.Submit(ctx, work, nonce, hash); err != nil || accepted {
		t.Fatalf("expected consumed job rejection, accepted=%v err=%v", accepted, err)
	}
	loaded, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(cfg.ChainFile, chain)
	if err != nil {
		t.Fatal(err)
	}
	if tip := loaded.TipSnapshot(); tip.Index != work.Height || hex.EncodeToString(tip.Hash) != hash || uint64(tip.Nonce) != nonce {
		t.Fatal("persisted block differs from submitted solution")
	}
}

func TestNormalMiningHTTPRequiresDurableCommit(t *testing.T) {
	h := externalMiningMux(t)
	chain, _ := nodeRuntime.Snapshot()
	cfg.ChainFile = t.TempDir()
	body, err := json.Marshal(MineRequest{Address: wallet.NewWallet().GetAddress()})
	if err != nil {
		t.Fatal(err)
	}
	response := webRequest(h, http.MethodPost, "/api/mine", string(body))
	if response.Code != http.StatusInternalServerError || chain.BestHeightSnapshot() != 0 {
		t.Fatalf("disk failure reached the live chain: status=%d height=%d", response.Code, chain.BestHeightSnapshot())
	}
}

func TestWebMiningFailureExpiryBindingAndCapacity(t *testing.T) {
	h := externalMiningMux(t)
	chain, _ := nodeRuntime.Snapshot()
	address := wallet.NewWallet().GetAddress()
	j := issueMiningJob(t, h, address)
	n, _ := solveMiningJob(t, j, 0)
	bad := strings.Replace(miningSubmitBody(j, n), address, wallet.NewWallet().GetAddress(), 1)
	if w := webRequest(h, "POST", "/api/mine/submit", bad); w.Code != 400 {
		t.Fatal("wrong miner accepted")
	}
	before, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	// A destination directory makes atomic installation fail deterministically.
	saved := cfg.ChainFile
	cfg.ChainFile = t.TempDir()
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(j, n)); w.Code != 409 {
		t.Fatal("disk failure accepted", w.Body.String())
	}
	cfg.ChainFile = saved
	after, _ := os.ReadFile(saved)
	if chain.BestHeightSnapshot() != 0 || !bytes.Equal(before, after) {
		t.Fatal("failed commit mutated chain")
	}
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(j, n)); w.Code != 200 {
		t.Fatal("retry failed", w.Body.String())
	}
	j = issueMiningJob(t, h, address)
	jobMu.Lock()
	webJobs[j.JobID].ExpiresAt = time.Now().Add(-time.Second)
	jobMu.Unlock()
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(j, 0)); w.Code != 409 {
		t.Fatal("expired accepted")
	}
	for i := 0; i < maxWebMiningJobsPerMiner; i++ {
		issueMiningJob(t, h, address)
	}
	if w := webRequest(h, "GET", "/api/mine/job?address="+address, ""); w.Code != 429 {
		t.Fatal("capacity not enforced", w.Code)
	}
	if w := webRequest(h, "GET", "/api/mine/job?address=invalid", ""); w.Code != 400 {
		t.Fatal("invalid address accepted")
	}
}

func TestWebMiningRejectsLegacyPuzzleInvalidPoWAndUnauthenticatedWork(t *testing.T) {
	h := externalMiningMux(t)
	address := wallet.NewWallet().GetAddress()
	j := issueMiningJob(t, h, address)
	request := httptest.NewRequest("GET", "/api/mine/job?address="+address, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != 401 {
		t.Fatal("job issuance without token", w.Code)
	}
	for _, nonce := range []string{`-1`, `"9223372036854775808"`, `1.5`, `"1e3"`} {
		body := `{"address":"` + address + `","job_id":"` + j.JobID + `","nonce":` + nonce + `}`
		if w := webRequest(h, "POST", "/api/mine/submit", body); w.Code != 400 {
			t.Fatal("invalid nonce accepted", nonce, w.Code)
		}
	}
	legacy := `{"address":"` + address + `","challenge":"` + j.JobID + `","nonce":0}`
	if w := webRequest(h, "POST", "/api/mine/submit", legacy); w.Code != 400 {
		t.Fatal("legacy fake puzzle accepted")
	}
	header, _ := hex.DecodeString(j.Header)
	var badNonce uint64
	for {
		binary.BigEndian.PutUint64(header[j.NonceOffset:], badNonce)
		sum := sha256.Sum256(header)
		if sum[0] != 0 {
			break
		}
		badNonce++
	}
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(j, badNonce)); w.Code != 409 {
		t.Fatal("invalid PoW accepted", w.Body.String())
	}
	chain, _ := nodeRuntime.Snapshot()
	if chain.BestHeightSnapshot() != 0 {
		t.Fatal("invalid work triggered server mining")
	}
	n, _ := solveMiningJob(t, j, 0)
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(j, n)); w.Code != 200 {
		t.Fatal("invalid attempt consumed work", w.Body.String())
	}
}

func TestMinerHTTPStartRegistersBeforeReplyAndStopReportsPending(t *testing.T) {
	h := externalMiningMux(t)
	t.Setenv("QC_NODE_DIR", t.TempDir())
	defer func() { nodeRuntime.StopMining(); nodeRuntime.WaitMining() }()
	address := wallet.NewWallet().GetAddress()
	w := webRequest(h, "POST", "/api/miner/start", `{"address":"`+address+`"}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if !nodeRuntime.IsMining() || nodeRuntime.MinerAddress() != address {
		t.Fatal("start reply preceded runtime registration")
	}
	w = webRequest(h, "POST", "/api/miner/stop", `{}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var status struct {
		Running  bool
		Stopping bool
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.Stopping || !status.Running {
		t.Fatal("stop response claimed completed stop", w.Body.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := nodeRuntime.WaitMiningContext(ctx); err != nil {
		t.Fatal(err)
	}
	if nodeRuntime.IsMining() {
		t.Fatal("loop retained after stop completion")
	}
}

func TestMiningAddressFallbackDoesNotGenerateOrWriteWallet(t *testing.T) {
	previousCfg, previousPaths := cfg, activeCLIPathsV1
	t.Cleanup(func() { cfg, activeCLIPathsV1 = previousCfg, previousPaths })
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfg = config.Default()
	cfg.WalletFile = filepath.Join(dir, "wallet_data.json")
	activeCLIPathsV1 = cliPathsV1{DataDir: dir}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	t.Setenv("APPDATA", dir)
	t.Setenv("QC_MINER", "")
	if addr := getDefaultAddress(); addr != "" {
		t.Fatal("address generated", addr)
	}
	if addr, err := ensureMinerAddress(); err == nil || addr != "" {
		t.Fatal("missing config accepted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("fallback wrote files")
	}
	addr := wallet.NewWallet().GetAddress()
	t.Setenv("QC_MINER", addr)
	cfg.Miner.Address = addr
	if got, err := ensureMinerAddress(); err != nil || got != addr {
		t.Fatal(got, err)
	}
	t.Setenv("QC_MINER", "invalid")
	cfg.Miner.Address = "invalid"
	if _, err := ensureMinerAddress(); err == nil {
		t.Fatal("invalid address accepted")
	}
	// Parse the actual default branch so future edits cannot silently re-enable
	// automatic mining or generate an unbacked wallet before argument dispatch.
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(original, "node_app_cli.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		x, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		cond, ok := x.Cond.(*ast.BinaryExpr)
		if !ok || cond.Op != token.LSS {
			return true
		}
		limit, ok := cond.Y.(*ast.BasicLit)
		if !ok || limit.Value != "2" {
			return true
		}
		call, ok := cond.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		fn, ok := call.Fun.(*ast.Ident)
		if !ok || fn.Name != "len" {
			return true
		}
		if len(call.Args) != 1 {
			return true
		}
		args, ok := call.Args[0].(*ast.Ident)
		if !ok || args.Name != "args" {
			return true
		}
		found = true
		ast.Inspect(x.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if id, ok := c.Fun.(*ast.Ident); ok && (id.Name == "startContinuousMining" || id.Name == "getDefaultAddress") {
					t.Error("default invocation starts mining or creates reward address")
				}
			}
			return true
		})
		return false
	})
	if !found {
		t.Fatal("default invocation branch missing")
	}
}
