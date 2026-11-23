package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	qnode "quantumcoin/node"
)

func TestIsolatedLocalMiningRequiresMatchingGenesis(t *testing.T) {
	_, _, owner, _ := signedAPIFixture(t)
	t.Setenv("QC_DIFFICULTY_BITS", "4")
	cfg.ChainFile = filepath.Join(t.TempDir(), config.QCIsolatedLocalChainFileV1)
	chain := blockchain.NewIsolatedLocalBlockchainV1(cfg.InitialReward, cfg.TotalSupply)
	if err := chain.BindRuntimeNetworkIdentityV1("local-release"); err != nil {
		t.Fatal(err)
	}
	nodeRuntime = qnode.NewRuntime(chain, cfg)
	if err := nodeRuntime.SaveChain(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	h := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	request := func(genesis string) *httptest.ResponseRecorder {
		body := strings.NewReader(`{"address":"` + owner.GetAddress() + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/mine/local-release", body)
		req.RemoteAddr = "127.0.0.1:32100"
		req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
		req.Header.Set("X-QC-Expected-Genesis", genesis)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	for _, genesis := range []string{"", strings.Repeat("0", 64)} {
		if w := request(genesis); w.Code != http.StatusConflict {
			t.Fatalf("wrong genesis status=%d body=%s", w.Code, w.Body.String())
		}
	}
	after, err := os.ReadFile(cfg.ChainFile)
	if err != nil || !bytes.Equal(before, after) || chain.BestHeightSnapshot() != 0 {
		t.Fatalf("rejected requests changed the chain: %v", err)
	}
	genesis := chain.NetworkIdentitySnapshotV1().GenesisHash
	filename := cfg.ChainFile
	cfg.ChainFile = filepath.Join(filename, config.QCIsolatedLocalChainFileV1)
	w := request(genesis)
	cfg.ChainFile = filename
	if w.Code != http.StatusInternalServerError || chain.BestHeightSnapshot() != 0 {
		t.Fatalf("disk failure published a block: status=%d body=%s", w.Code, w.Body.String())
	}
	w = request(genesis)
	if w.Code != http.StatusOK || chain.BestHeightSnapshot() != 1 {
		t.Fatalf("pinned mining failed: status=%d body=%s", w.Code, w.Body.String())
	}
	saved, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(cfg.ChainFile, chain)
	if err != nil || saved.BestHeightSnapshot() != 1 {
		t.Fatalf("accepted block did not persist: %v", err)
	}
}
