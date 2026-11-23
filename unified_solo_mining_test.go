package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/miner"
	qnode "quantumcoin/node"
	"quantumcoin/wallet"
)

// Use the published fresh testnet, without pre-mining or disabling maturity.
func unifiedSoloMux(t *testing.T) http.Handler {
	t.Helper()
	previous := config.Current()
	t.Cleanup(func() { config.Set(previous) })
	readOnlyMuxFixtureV1(t)
	t.Setenv("QC_NETWORK", "testnet-local")
	t.Setenv("QC_DIFFICULTY_BITS", "")
	var err error
	cfg, err = config.LoadWithDataDir(filepath.Join("release", "config.testnet.json"), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	chain, err := blockchain.NewNetworkBlockchainV4(cfg, config.QCNetworkProfileTestnetLocalV1)
	if err != nil {
		t.Fatal(err)
	}
	chain.SetCoinbaseMaturity(10) // match the fresh-network bootstrap policy
	if err = chain.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	nodeRuntime = qnode.NewRuntime(chain, cfg)
	apiReadOnlyMode = false
	webJobs = map[string]*webJob{}
	return newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
}

func TestUnifiedSoloNativeMinerReconcilesLostResponseWithoutResubmission(t *testing.T) {
	h := unifiedSoloMux(t)
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/mine/submit" {
			h.ServeHTTP(w, r)
			return
		}
		posts++
		committed := httptest.NewRecorder()
		h.ServeHTTP(committed, r)
		if committed.Code != 200 {
			t.Errorf("commit failed: %s", committed.Body.String())
		}
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()
	genesis := nodeRuntime.MiningTip().GenesisHash
	backend, err := miner.NewPinnedNodeBackend(server.URL, actualMuxTestTokenV1, genesis)
	if err != nil {
		t.Fatal(err)
	}
	address := wallet.NewWallet().GetAddress()
	work, err := backend.GetWork(context.Background(), address)
	if err != nil {
		t.Fatal(err)
	}
	job := WebMineJobResp{Header: hex.EncodeToString(work.Header), NonceOffset: len(work.Header) - 8, Protocol: webMiningProtocol}
	nonce, hash := solveMiningJob(t, job, 0)
	accepted, err := backend.Submit(context.Background(), work, nonce, hash)
	if !accepted || err != nil || posts != 1 {
		t.Fatal("lost response reconciliation failed", accepted, err, posts)
	}
	chain, _ := nodeRuntime.Snapshot()
	if chain.TotalMintedSnapshot() != 50 || chain.BestHeightSnapshot() != 1 {
		t.Fatal("retry created extra block")
	}
	foreign, err := miner.NewPinnedNodeBackend(server.URL, actualMuxTestTokenV1, strings.Repeat("aa", 32))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = foreign.GetWork(context.Background(), address); err == nil {
		t.Fatal("native client accepted foreign genesis")
	}
}

func TestUnifiedSoloMiningPCAndTelegramOneRewardAndDurableReceipt(t *testing.T) {
	h := unifiedSoloMux(t)
	chain, _ := nodeRuntime.Snapshot()
	genesis := chain.MiningTipSnapshot()
	if genesis.Height != 0 || chain.TotalMintedSnapshot() != 0 {
		t.Fatal("fixture must start at empty genesis")
	}
	jobs := make([]WebMineJobResp, 100)
	for i := range jobs {
		jobs[i] = issueMiningJob(t, h, wallet.NewWallet().GetAddress())
		if jobs[i].Height != 1 || jobs[i].PrevHash != genesis.Hash || jobs[i].GenesisHash != genesis.GenesisHash {
			t.Fatal("clients did not receive common tip")
		}
	}
	// PC wins block one; the 75 Telegram participants receive no second reward.
	winner := jobs[0]
	nonce, hash := solveMiningJob(t, winner, 0)
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(winner, nonce)); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if chain.TotalMintedSnapshot() != 50 {
		t.Fatal("reward multiplied by participants")
	}
	block := chain.BlockByIndexSnapshot(1)
	if block.Miner != winner.Miner || len(block.Transactions) != 1 || len(block.Transactions[0].Outputs) != 1 || block.Transactions[0].Outputs[0].Amount != 50 {
		t.Fatal("wrong solo coinbase", block)
	}
	for _, job := range jobs[1:] {
		if chain.BalanceSnapshot(job.Miner) != 0 {
			t.Fatal("non-winning participant credited")
		}
	}
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(winner, nonce)); w.Code != 409 {
		t.Fatal("replayed proof accepted", w.Code)
	}
	stale := jobs[25]
	staleNonce, _ := solveMiningJob(t, stale, 0)
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(stale, staleNonce)); w.Code != 409 {
		t.Fatal("old tip accepted", w.Code, w.Body.String())
	}
	// Telegram wins block two using exactly the same work and commit contract.
	next := issueMiningJob(t, h, jobs[25].Miner)
	if next.Height != 2 || next.PrevHash != hash {
		t.Fatal("late joining miner missed current tip")
	}
	n2, h2 := solveMiningJob(t, next, 0)
	if w := webRequest(h, "POST", "/api/mine/submit", miningSubmitBody(next, n2)); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if chain.TotalMintedSnapshot() != 100 || chain.BlockByIndexSnapshot(2).Miner != next.Miner {
		t.Fatal("Telegram reward differs from solo reward")
	}
	loaded, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(cfg.ChainFile, chain)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := hex.DecodeString(h2)
	if loaded.TotalMintedSnapshot() != 100 || !loaded.MiningResultSnapshot(2, raw, next.Miner) || loaded.MiningResultSnapshot(2, raw, winner.Miner) {
		t.Fatal("receipt did not survive canonical disk replay")
	}
	path := fmt.Sprintf("/api/mine/result?height=2&hash=%s&address=%s", h2, next.Miner)
	response := webRequest(h, "GET", path, "")
	var result struct {
		Accepted bool `json:"accepted"`
	}
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &result) != nil || !result.Accepted {
		t.Fatal("receipt lookup failed", response.Body.String())
	}
}

func TestUnifiedSoloMiningTipAndReceiptRequireAuthAndAreReadOnly(t *testing.T) {
	h := unifiedSoloMux(t)
	for _, path := range []string{"/api/mine/tip", "/api/mine/result?height=1&hash=" + strings.Repeat("00", 32) + "&address=" + wallet.NewWallet().GetAddress()} {
		if w := readOnlyRequestV1(h, "GET", path, ""); w.Code != 401 {
			t.Fatal("missing token accepted", path, w.Code)
		}
		apiReadOnlyMode = true
		nodeRuntime.SetAPIReadOnly(true)
		if w := webRequest(h, "GET", path, ""); w.Code != 200 {
			t.Fatal("read-only lookup denied", path, w.Code, w.Body.String())
		}
		if w := webRequest(h, "POST", path, ""); w.Code < 400 {
			t.Fatal("mutating method accepted")
		}
	}
}
