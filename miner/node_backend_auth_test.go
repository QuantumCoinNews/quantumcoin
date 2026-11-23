package miner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

func TestNodeWorkerCredentialsAndDeniedPreflight(t *testing.T) {
	genesis, address := strings.Repeat("ab", 32), wallet.NewWallet().GetAddress()
	for _, mode := range []string{"allowed", "unauthorized", "forbidden", "missing-token"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("X-QC-API-KEY") != "worker-fixture-token" || r.URL.Query().Has("token") {
					t.Error("worker credential transport mismatch")
				}
				if mode == "unauthorized" {
					http.Error(w, "denied", http.StatusUnauthorized)
					return
				}
				if mode == "forbidden" {
					http.Error(w, "denied", http.StatusForbidden)
					return
				}
				if r.URL.Path == "/api/mine/job" {
					if r.URL.Query().Get("address") != address {
						t.Error("worker reward address changed")
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"genesis_hash": genesis, "prev_hash": strings.Repeat("cd", 32), "job_id": strings.Repeat("aa", 32), "header": strings.Repeat("00", 16), "expires": time.Now().Add(30 * time.Second).Unix(), "protocol": "QC_WEB_BLOCK_POW_V1", "nonce_offset": 8, "nonce_encoding": "uint64-big-endian", "max_nonce": strconv.FormatUint(blockchain.MaxConsensusNonceV1, 10), "difficulty": 255, "miner": address, "height": 1})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{})
			}))
			defer server.Close()
			token := "worker-fixture-token"
			if mode == "missing-token" {
				token = ""
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			worker, err := PrepareNodeWorker(ctx, server.URL, token, genesis, WorkerConfig{Address: address, Threads: 1})
			if mode != "allowed" {
				if worker != nil || err == nil {
					t.Fatal("denied preflight created a worker")
				}
				if mode != "missing-token" && !errors.Is(err, errNodeMiningDenied) {
					t.Fatal(err)
				}
				want := int32(1)
				if mode == "missing-token" {
					want = 0
				}
				if calls.Load() != want {
					t.Fatal("unexpected auth retries")
				}
				return
			}
			if err != nil || worker == nil || worker.initial == nil {
				t.Fatal("valid authenticated preflight failed", err)
			}
			// Exercise the shared credential boundary for every subsequent worker route;
			// no hashing or claimed consensus acceptance is involved in this transport test.
			backend := worker.backend.(*nodeBackend)
			for _, route := range []struct{ method, path string }{{http.MethodGet, "/api/mine/tip"}, {http.MethodPost, "/api/mine/submit"}, {http.MethodGet, "/api/mine/result"}} {
				var result map[string]any
				if status, err := backend.request(ctx, route.method, route.path, nil, &result); err != nil || status != http.StatusOK {
					t.Fatal("worker route lost authentication", err)
				}
			}
			if calls.Load() != 4 {
				t.Fatal("worker credential requests not covered")
			}
		})
	}
}
