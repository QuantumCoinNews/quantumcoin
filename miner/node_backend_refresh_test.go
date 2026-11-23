package miner

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNodeBackendDetectsSameHeightReorgAndDifferentGenesis(t *testing.T) {
	genesis, previous := strings.Repeat("ab", 32), strings.Repeat("cd", 32)
	var changed, foreign atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-QC-API-KEY") != "test-only-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		hash := previous
		if changed.Load() {
			hash = strings.Repeat("ef", 32)
		}
		g := genesis
		if foreign.Load() {
			g = strings.Repeat("aa", 32)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"version": "QC_MINING_TIP_V1", "height": 5, "hash": hash, "genesis_hash": g})
	}))
	defer server.Close()
	backend, err := NewPinnedNodeBackend(server.URL, "test-only-token", genesis)
	if err != nil {
		t.Fatal(err)
	}
	source := backend.(*nodeBackend)
	work := &Work{Height: 6, PrevHash: previous, GenesisHash: genesis, ExpiresAt: time.Now().Add(time.Minute)}
	current, err := source.WorkCurrent(context.Background(), work)
	if !current || err != nil {
		t.Fatal(current, err)
	}
	changed.Store(true)
	current, err = source.WorkCurrent(context.Background(), work)
	if current || err != nil {
		t.Fatal("equal-height reorganization missed", current, err)
	}
	foreign.Store(true)
	current, err = source.WorkCurrent(context.Background(), work)
	if current || !errors.Is(err, errNodeMiningDenied) {
		t.Fatal("foreign genesis not fatal", current, err)
	}
}
