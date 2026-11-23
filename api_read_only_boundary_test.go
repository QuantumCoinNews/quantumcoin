package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"quantumcoin/ai"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	qnode "quantumcoin/node"
	"quantumcoin/wallet"
)

func readOnlyMuxFixtureV1(t *testing.T) http.Handler {
	t.Helper()
	clearActualMuxTestEnvV1(t)
	t.Setenv("QC_API_TOKEN", actualMuxTestTokenV1)
	t.Setenv("QC_DEV_MODE", "1")
	oldRuntime, oldConfig := nodeRuntime, cfg
	oldMode, oldJobs := apiReadOnlyMode, webJobs
	t.Cleanup(func() {
		nodeRuntime.StopMining()
		nodeRuntime, cfg = oldRuntime, oldConfig
		apiReadOnlyMode, webJobs = oldMode, oldJobs
	})
	cfg = config.Default()
	cfg.ChainFile = filepath.Join(t.TempDir(), "chain_data.dat")
	bc := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	if err := bc.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	nodeRuntime = qnode.NewRuntime(bc, cfg)
	nodeRuntime.SetAPIReadOnly(true)
	apiReadOnlyMode = true
	webJobs = map[string]*webJob{"existing-job": {Miner: "existing-miner"}}
	return newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
}

func readOnlyRequestV1(handler http.Handler, method, path, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(`{"amount":"1"}`))
	request.RemoteAddr = "127.0.0.1:4275"
	request.Host = "127.0.0.1:8082"
	request.Header.Set("X-QC-API-KEY", token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestReadOnlyMuxDeniesEveryMutatingCapabilityWithoutSideEffects(t *testing.T) {
	handler := readOnlyMuxFixtureV1(t)
	bc, _ := nodeRuntime.Snapshot()
	beforeFile, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := func() []byte {
		data, err := json.Marshal([]any{bc.AllBlocksSnapshot(), bc.PendingTransactionsSnapshot(), cfg, ai.GetAlertsSnapshot(), webJobs})
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	beforeState := snapshot()
	beforeEntries, err := os.ReadDir(filepath.Dir(cfg.ChainFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/api/tx/send", "/api/mine", "/api/mine/local-release", "/api/dev/fastmine", "/api/tx/burn",
		"/api/mine/submit", "/api/miner/start", "/api/miner/stop",
		"/api/game/score?player=attacker&score=100000", "/api/mine/job?address=attacker",
		"/api/wallet/new", "/api/stake/start", "/api/nft/monthly/mint",
	} {
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "TRACE"} {
			t.Run(method+" "+path, func(t *testing.T) {
				recorder := readOnlyRequestV1(handler, method, path, actualMuxTestTokenV1)
				if recorder.Code != http.StatusForbidden {
					t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
				}
				var payload struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil || payload.Success || payload.Message != "API mode is read-only" {
					t.Fatalf("unexpected denial: %s (%v)", recorder.Body.String(), err)
				}
			})
		}
	}
	afterFile, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeFile, afterFile) || !bytes.Equal(beforeState, snapshot()) || nodeRuntime.IsMining() {
		t.Fatal("denied requests changed chain, config, mempool, game, alerts, mining job or miner state")
	}
	files, err := os.ReadDir(filepath.Dir(cfg.ChainFile))
	if err != nil || len(files) != len(beforeEntries) {
		t.Fatalf("denied request created a file: %v %v", files, err)
	}
	for i, file := range files {
		if file.Name() != beforeEntries[i].Name() {
			t.Fatal("denied request changed directory entries")
		}
	}
}

func TestReadOnlyMuxPreservesReadsAndHeaderAuthentication(t *testing.T) {
	handler := readOnlyMuxFixtureV1(t)
	for _, path := range []string{"/api/health", "/api/economy/policy", "/api/blocks", "/api/miner/status", "/api/ai/alerts"} {
		t.Run(path, func(t *testing.T) {
			want := http.StatusOK
			if r := readOnlyRequestV1(handler, "GET", path, actualMuxTestTokenV1); r.Code != want {
				t.Fatalf("read status=%d body=%s", r.Code, r.Body.String())
			}
			if r := readOnlyRequestV1(handler, "GET", path, ""); r.Code != http.StatusUnauthorized {
				t.Fatalf("unauthenticated read status=%d", r.Code)
			}
			if r := readOnlyRequestV1(handler, "POST", path, actualMuxTestTokenV1); r.Code != http.StatusForbidden {
				t.Fatalf("non-read method status=%d", r.Code)
			}
		})
	}
	for _, token := range []string{"", "wrong"} {
		if r := readOnlyRequestV1(handler, "POST", "/api/miner/start", token); r.Code != http.StatusUnauthorized {
			t.Fatalf("denied capability bypassed auth: %d", r.Code)
		}
	}
	// Receipt POST reaches its read handler; malformed evidence is rejected by
	// that handler, rather than being mistaken for a forbidden mint operation.
	if r := readOnlyRequestV1(handler, "POST", "/api/nft/monthly/receipt", actualMuxTestTokenV1); r.Code != http.StatusBadRequest {
		t.Fatalf("signed receipt read was blocked: %d %s", r.Code, r.Body.String())
	}
}

func TestReadOnlyBoundaryFailsClosedForFutureAndDisguisedRoutes(t *testing.T) {
	readOnlyMuxFixtureV1(t)
	called := false
	handler := withAPIReadOnlyBoundaryV1(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) }))
	for _, path := range []string{"/api/future/write", "/api/game/score/", "/api/%67ame/score", "/api//game/score", "/api/mine/../game/score", "/api/miner/start?method=GET", "/api/health/../game/score"} {
		if r := readOnlyRequestV1(handler, "GET", path, actualMuxTestTokenV1); r.Code != http.StatusForbidden || called {
			t.Fatalf("route reached downstream handler: %s status=%d", path, r.Code)
		}
	}
	// The startup restriction survives a later reset of compatibility globals.
	apiReadOnlyMode = false
	nodeRuntime.SetAPIReadOnly(false)
	if r := readOnlyRequestV1(handler, "POST", "/api/tx/send", actualMuxTestTokenV1); r.Code != http.StatusForbidden || called {
		t.Fatal("read-only server was downgraded")
	}
}

func TestReadOnlyBoundaryAlsoHonorsRuntimeMode(t *testing.T) {
	readOnlyMuxFixtureV1(t)
	apiReadOnlyMode = false
	nodeRuntime.SetAPIReadOnly(false)
	called := false
	handler := withAPIReadOnlyBoundaryV1(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) }))
	nodeRuntime.SetAPIReadOnly(true)
	if r := readOnlyRequestV1(handler, "POST", "/api/tx/send", actualMuxTestTokenV1); r.Code != http.StatusForbidden || called {
		t.Fatal("runtime read-only mode ignored")
	}
}

func TestWritableBoundaryPreservesExistingDispatch(t *testing.T) {
	readOnlyMuxFixtureV1(t)
	apiReadOnlyMode = false
	nodeRuntime.SetAPIReadOnly(false)
	var received []string
	handler := withAPIReadOnlyBoundaryV1(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = append(received, r.Method)
		w.WriteHeader(204)
	}))
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		if r := readOnlyRequestV1(handler, method, "/api/game/score", actualMuxTestTokenV1); r.Code != 204 {
			t.Fatalf("writable dispatch changed: %d", r.Code)
		}
	}
	if !reflect.DeepEqual(received, []string{"GET", "POST", "PUT", "DELETE"}) {
		t.Fatal(received)
	}
}

func TestReadOnlyWalletAddressNeverCreatesMissingWallet(t *testing.T) {
	handler := readOnlyMuxFixtureV1(t)
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("QC_MINER", "")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	})
	if r := readOnlyRequestV1(handler, "GET", "/api/wallet/address", actualMuxTestTokenV1); r.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing address status=%d body=%s", r.Code, r.Body.String())
	}
	if files, err := os.ReadDir(directory); err != nil || len(files) != 0 {
		t.Fatalf("address read generated a file: %v %v", files, err)
	}
	address := wallet.NewWallet().GetAddress()
	t.Setenv("QC_MINER", address)
	cfg.Miner.Address = address
	r := readOnlyRequestV1(handler, "GET", "/api/wallet/address", actualMuxTestTokenV1)
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), address) {
		t.Fatalf("existing address unavailable: %d %s", r.Code, r.Body.String())
	}
}

func TestReadOnlyMuxPreflightDoesNotDispatchMutation(t *testing.T) {
	handler := readOnlyMuxFixtureV1(t)
	r := readOnlyRequestV1(handler, "OPTIONS", "/api/game/score?player=attacker&score=10", "")
	if r.Code != http.StatusNoContent || r.Body.Len() != 0 {
		t.Fatalf("unsafe preflight: %d", r.Code)
	}
}

func TestWritableMuxRejectsLegacyGETGameScoreMutation(t *testing.T) {
	readOnlyMuxFixtureV1(t)
	apiReadOnlyMode = false
	nodeRuntime.SetAPIReadOnly(false)
	handler := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	r := readOnlyRequestV1(handler, "GET", "/api/game/score?player=existing-player&score=9", actualMuxTestTokenV1)
	if r.Code != http.StatusNotFound || r.Header().Get("Allow") != "" {
		t.Fatalf("writable game route changed: status=%d allow=%q", r.Code, r.Header().Get("Allow"))
	}
}
