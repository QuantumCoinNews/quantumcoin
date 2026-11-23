package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"quantumcoin/blockchain"
	qnode "quantumcoin/node"
	"quantumcoin/wallet"
)

func TestMiningRouteUnavailableIsNotMissingBlock(t *testing.T) {
	externalMiningMux(t)
	for _, nilRuntime := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-chain", true: "missing-runtime"}[nilRuntime], func(t *testing.T) {
			previous := nodeRuntime
			defer func() { nodeRuntime = previous }()
			nodeRuntime = qnode.NewRuntime(nil, cfg)
			if nilRuntime {
				nodeRuntime = nil
			}
			handler := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
			for _, path := range []string{"/api/block?index=0", "/api/block?hash=" + strings.Repeat("00", 32)} {
				w := webRequest(handler, "GET", path, "")
				if w.Code != 503 {
					t.Errorf("%s status=%d want=503: %s", path, w.Code, w.Body.String())
				}
			}
			w := webRequest(handler, "POST", "/api/mine", `{"address":"`+wallet.NewWallet().GetAddress()+`"}`)
			if w.Code != 503 {
				t.Errorf("mine unavailable status=%d want=503", w.Code)
			}
		})
	}
}

func TestMiningRouteDenialsPreserveChainAndJobs(t *testing.T) {
	handler := externalMiningMux(t)
	chain, _ := nodeRuntime.Snapshot()
	fileBefore, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	stateBefore, err := json.Marshal([]any{chain.AllBlocksSnapshot(), chain.PendingTransactionsSnapshot(), webJobs})
	if err != nil {
		t.Fatal(err)
	}
	address := wallet.NewWallet().GetAddress()
	for _, row := range []struct {
		name, method, path, token, body string
		readonly                        bool
		status                          int
	}{
		{"block-auth", "GET", "/api/block?index=0", "", "", false, 401},
		{"mine-auth", "POST", "/api/mine", "", `{"address":"` + address + `"}`, false, 401},
		{"block-method", "POST", "/api/block", actualMuxTestTokenV1, "", false, 405},
		{"mine-method", "GET", "/api/mine", actualMuxTestTokenV1, "", false, 405},
		{"missing-selector", "GET", "/api/block", actualMuxTestTokenV1, "", false, 400},
		{"invalid-index", "GET", "/api/block?index=not-a-number", actualMuxTestTokenV1, "", false, 400},
		{"missing-block", "GET", "/api/block?index=99", actualMuxTestTokenV1, "", false, 404},
		{"invalid-hash", "GET", "/api/block?hash=not-hex", actualMuxTestTokenV1, "", false, 400},
		{"missing-hash", "GET", "/api/block?hash=" + strings.Repeat("00", 32), actualMuxTestTokenV1, "", false, 404},
		{"invalid-json", "POST", "/api/mine", actualMuxTestTokenV1, `{`, false, 400},
		{"missing-miner", "POST", "/api/mine", actualMuxTestTokenV1, `{}`, false, 400},
		{"read-only-mine", "POST", "/api/mine", actualMuxTestTokenV1, `{"address":"` + address + `"}`, true, 403},
	} {
		t.Run(row.name, func(t *testing.T) {
			nodeRuntime.SetAPIReadOnly(row.readonly)
			defer nodeRuntime.SetAPIReadOnly(false)
			req := httptest.NewRequest(row.method, row.path, strings.NewReader(row.body))
			req.RemoteAddr = "127.0.0.1:34631"
			req.Host = "127.0.0.1:8082"
			req.Header.Set("X-QC-API-KEY", row.token)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != row.status {
				t.Fatalf("status=%d want=%d: %s", w.Code, row.status, w.Body.String())
			}
			var failure struct {
				Success bool
				Message string
			}
			if json.Unmarshal(w.Body.Bytes(), &failure) != nil || failure.Success || failure.Message == "" {
				t.Fatalf("invalid error envelope: %s", w.Body.String())
			}
			after, err := json.Marshal([]any{chain.AllBlocksSnapshot(), chain.PendingTransactionsSnapshot(), webJobs})
			if err != nil {
				t.Fatal(err)
			}
			fileAfter, err := os.ReadFile(cfg.ChainFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(after, stateBefore) || !bytes.Equal(fileBefore, fileAfter) {
				t.Fatal("rejected request changed chain/mempool/jobs/file")
			}
		})
	}
}

func TestMiningRouteDurableReceiptAndExplorerOverTCP(t *testing.T) {
	handler := externalMiningMux(t)
	chain, _ := nodeRuntime.Snapshot()
	server := httptest.NewServer(handler)
	defer server.Close()
	client := &http.Client{Timeout: 10 * time.Second}
	request := func(method, path, body string) []byte {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != 200 {
			t.Fatalf("%s %s status=%d: %s", method, path, response.StatusCode, data)
		}
		if !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
			t.Fatal("response is not JSON")
		}
		return data
	}
	address := wallet.NewWallet().GetAddress()
	var receipt struct {
		Success bool   `json:"success"`
		Height  int    `json:"height"`
		Hash    string `json:"block_hash"`
		Reward  int    `json:"reward"`
	}
	if err := json.Unmarshal(request("POST", "/api/mine", `{"address":"`+address+`"}`), &receipt); err != nil {
		t.Fatal(err)
	}
	loaded, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(cfg.ChainFile, chain)
	if err != nil {
		t.Fatal(err)
	}
	block := loaded.TipSnapshot()
	if !receipt.Success || receipt.Height != 1 || block.Index != receipt.Height || receipt.Hash != hex.EncodeToString(block.Hash) || receipt.Reward != chain.ExpectedSubsidyForHeightSnapshot(block.Index) {
		t.Fatalf("receipt does not match durable block: %+v", receipt)
	}
	if len(block.Transactions) == 0 || len(block.Transactions[0].Outputs) == 0 || !bytes.Equal(block.Transactions[0].Outputs[0].PubKeyHash, wallet.Base58DecodeAddress(address)) {
		t.Fatal("reward not assigned to requested miner")
	}
	nodeRuntime.SetAPIReadOnly(true)
	for _, selector := range []string{"index=1", "hash=" + receipt.Hash} {
		var detail explorerBlockV1
		if err := json.Unmarshal(request("GET", "/api/block?"+selector, ""), &detail); err != nil {
			t.Fatal(err)
		}
		if detail.Index != block.Index || !bytes.Equal(detail.Hash, block.Hash) {
			t.Fatalf("explorer does not identify committed block: %+v", detail)
		}
	}
}
