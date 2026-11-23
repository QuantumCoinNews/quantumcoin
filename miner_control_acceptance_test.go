package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	qnode "quantumcoin/node"
	"quantumcoin/security"
	"quantumcoin/wallet"
)

func frozenMinerControlFixtureV1(t *testing.T) (*qnode.Runtime, string, func()) {
	t.Helper()
	readOnlyMuxFixtureV1(t)
	t.Setenv("QC_NETWORK", "testnet-local")
	t.Setenv("QC_NET_MAGIC", config.QCTestnetMagicV1)
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	cfg.DefaultDifficultyBits = config.QCTestnetDefaultDifficultyBitsV1
	cfg.HTTPPort = ":18082"
	cfg.P2PPort = ":13001"
	cfg.ChainFile = filepath.Join(filepath.Dir(cfg.ChainFile), config.QCTestnetChainFileV1)
	chain, err := blockchain.NewNetworkBlockchainV4(cfg, config.QCNetworkProfileTestnetLocalV1)
	if err != nil {
		t.Fatal(err)
	}
	if err = chain.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	if err = security.ConfigureEventDirectory(cfg.ChainFile, "testnet-local"); err != nil {
		t.Fatal(err)
	}
	nodeRuntime = qnode.NewRuntime(chain, cfg)
	apiReadOnlyMode = false
	nodeRuntime.SetAPIReadOnly(false)
	r := nodeRuntime
	address := wallet.NewWallet().GetAddress()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	if err := r.StartMining(address, qnode.MiningHooksV1{Difficulty: func() int { close(entered); <-release; return 255 }}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		r.StopMining()
		unblock()
		done := make(chan struct{})
		go func() { r.WaitMining(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("test miner failed to stop")
		}
	})
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("test miner did not enter controlled difficulty hook")
	}
	return r, address, unblock
}

func minerControlHTTPV1(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, "http://127.0.0.1:8082"+path, strings.NewReader(body))
	request.RemoteAddr = "127.0.0.1:4275"
	request.Host = "127.0.0.1:8082"
	request.Header.Set("X-QC-API-KEY", actualMuxTestTokenV1)
	request.Header.Set("Content-Type", "application/json")
	return request
}

func TestMinerControlRejectsInvalidActiveAddressAndForwardedHost(t *testing.T) {
	r, address, _ := frozenMinerControlFixtureV1(t)
	for _, mode := range []string{"invalid-active-address", "forwarded-host"} {
		t.Run(mode, func(t *testing.T) {
			selected := address
			if mode == "invalid-active-address" {
				selected = "invalid-address"
			}
			raw, _ := json.Marshal(minerStartReq{Address: selected})
			request := minerControlHTTPV1(http.MethodPost, "/api/miner/start", string(raw))
			want := http.StatusBadRequest
			if mode == "forwarded-host" {
				request.Header.Set("X-Forwarded-Host", "external.example")
				want = http.StatusForbidden
			}
			recorder := httptest.NewRecorder()
			newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(recorder, request)
			if recorder.Code != want {
				t.Fatalf("unvalidated control accepted: status=%d want=%d", recorder.Code, want)
			}
			if !r.IsMining() || r.MinerAddress() != address {
				t.Fatal("denied request affected existing miner")
			}
		})
	}
}

func assertMinerControlFileUnchangedV1(t *testing.T, before []byte) {
	t.Helper()
	after, err := os.ReadFile(cfg.ChainFile)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("denied control changed persisted chain")
	}
}

func TestMinerControlDeniedRequestsPreserveActiveMiner(t *testing.T) {
	r, address, _ := frozenMinerControlFixtureV1(t)
	before, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		want   int
		mutate func(*testing.T, *http.Request)
	}{
		{"missing-token", http.StatusUnauthorized, func(t *testing.T, r *http.Request) { r.Header.Del("X-QC-API-KEY") }},
		{"wrong-token", http.StatusUnauthorized, func(t *testing.T, r *http.Request) { r.Header.Set("X-QC-API-KEY", "wrong-fixture-token") }},
		{"query-token", http.StatusUnauthorized, func(t *testing.T, r *http.Request) {
			r.Header.Del("X-QC-API-KEY")
			r.URL.RawQuery = "token=" + actualMuxTestTokenV1
		}},
		{"unconfigured-token", http.StatusUnauthorized, func(t *testing.T, r *http.Request) { t.Setenv("QC_API_TOKEN", "") }},
		{"get", http.StatusMethodNotAllowed, func(t *testing.T, r *http.Request) { r.Method = http.MethodGet }},
		{"put", http.StatusMethodNotAllowed, func(t *testing.T, r *http.Request) { r.Method = http.MethodPut }},
		{"cross-origin", http.StatusForbidden, func(t *testing.T, r *http.Request) { r.Header.Set("Origin", "https://external.example") }},
		{"fetch-site", http.StatusForbidden, func(t *testing.T, r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }},
		{"external-host", http.StatusForbidden, func(t *testing.T, r *http.Request) { r.Host = "external.example" }},
		{"remote-client", http.StatusForbidden, func(t *testing.T, r *http.Request) { r.RemoteAddr = "203.0.113.5:3456" }},
		{"no-dev-mode", http.StatusForbidden, func(t *testing.T, r *http.Request) { t.Setenv("QC_DEV_MODE", "") }},
		{"public-testnet", http.StatusForbidden, func(t *testing.T, r *http.Request) { t.Setenv("QC_PUBLIC_TESTNET", "1") }},
		{"public-bind", http.StatusForbidden, func(t *testing.T, r *http.Request) {
			listen := "0.0.0.0" + getHTTPAddr()
			t.Setenv("QC_API_LISTEN", listen)
			if getAPIListenAddr() != listen {
				t.Fatal("fixture did not select the exposed listener")
			}
		}},
		{"read-only", http.StatusForbidden, func(t *testing.T, request *http.Request) {
			r.SetAPIReadOnly(true)
			t.Cleanup(func() { r.SetAPIReadOnly(false) })
		}},
	}
	for _, header := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Forwarded-Port", "X-Real-IP", "Via"} {
		name := header
		cases = append(cases, struct {
			name   string
			want   int
			mutate func(*testing.T, *http.Request)
		}{"proxy-" + name, http.StatusForbidden, func(t *testing.T, r *http.Request) { r.Header.Set(name, "proxy-fixture") }})
	}
	raw, _ := json.Marshal(minerStartReq{Address: address})
	for _, path := range []string{"/api/miner/start", "/api/miner/stop"} {
		t.Run(path, func(t *testing.T) {
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					request := minerControlHTTPV1(http.MethodPost, path, string(raw))
					c.mutate(t, request)
					response := httptest.NewRecorder()
					newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(response, request)
					if response.Code != c.want {
						t.Fatalf("status=%d want=%d", response.Code, c.want)
					}
					if !r.IsMining() || r.MinerAddress() != address {
						t.Fatal("denied request changed miner ownership or stopped it")
					}
					assertMinerControlFileUnchangedV1(t, before)
				})
			}
		})
	}
}

func TestMinerControlRequiresUnambiguousExplicitStartAddress(t *testing.T) {
	r, address, _ := frozenMinerControlFixtureV1(t)
	cfg.Miner.Address = address // A configured fallback must not rescue invalid input.
	before, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"", `{`, `null`, `{}`, `{"address":null}`, `{"address":42}`, `{"address":"invalid-address"}`,
		`{"address":"` + address + `","extra":true}`,
		`{"address":"` + address + `","address":"` + address + `"}`,
		`{"address":"` + address + `","ADDRESS":"` + address + `"}`,
		`{"address":"` + address + `"} {}`,
	} {
		t.Run(body, func(t *testing.T) {
			request := minerControlHTTPV1(http.MethodPost, "/api/miner/start", body)
			response := httptest.NewRecorder()
			newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("invalid start intent accepted: %d", response.Code)
			}
			if !r.IsMining() || r.MinerAddress() != address {
				t.Fatal("invalid start changed existing miner")
			}
			assertMinerControlFileUnchangedV1(t, before)
		})
	}
	for _, body := range []string{`{"address":"` + address + `"}`, strings.Repeat(" ", int(apiMaxBodyBytes)+1)} {
		request := minerControlHTTPV1(http.MethodPost, "/api/miner/start", body)
		want := http.StatusRequestEntityTooLarge
		if len(body) < int(apiMaxBodyBytes) {
			request.URL.RawQuery = "address=" + address
			want = http.StatusBadRequest
		}
		response := httptest.NewRecorder()
		newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(response, request)
		if response.Code != want {
			t.Fatalf("query/size boundary status=%d want=%d", response.Code, want)
		}
	}
}

func TestMinerControlAuthorizedSameAddressAndStop(t *testing.T) {
	r, address, unblock := frozenMinerControlFixtureV1(t)
	for _, selected := range []string{address, wallet.NewWallet().GetAddress()} {
		raw, _ := json.Marshal(minerStartReq{Address: selected})
		request := minerControlHTTPV1(http.MethodPost, "/api/miner/start", string(raw))
		response := httptest.NewRecorder()
		newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(response, request)
		want := http.StatusOK
		if selected != address {
			want = http.StatusConflict
		}
		if response.Code != want || r.MinerAddress() != address {
			t.Fatalf("active miner address was replaced or hidden: %d", response.Code)
		}
	}
	request := minerControlHTTPV1(http.MethodGet, "/api/miner/status", "")
	response := httptest.NewRecorder()
	newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatal("authorized status rejected")
	}
	request = minerControlHTTPV1(http.MethodPost, "/api/miner/stop", "")
	request.Header.Del("X-QC-API-KEY")
	request.Header.Set("Authorization", "Bearer "+actualMuxTestTokenV1)
	response = httptest.NewRecorder()
	newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatal("authorized stop rejected")
	}
	unblock()
	done := make(chan struct{})
	go func() { r.WaitMining(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("authorized stop did not finish")
	}
	if r.IsMining() {
		t.Fatal("authorized stop left miner running")
	}
}

func TestMinerControlAuthorizedFreshStart(t *testing.T) {
	r, _, unblock := frozenMinerControlFixtureV1(t)
	r.StopMining()
	unblock()
	done := make(chan struct{})
	go func() { r.WaitMining(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("fixture did not stop")
	}
	address := wallet.NewWallet().GetAddress()
	raw, _ := json.Marshal(minerStartReq{Address: address})
	request := minerControlHTTPV1(http.MethodPost, "/api/miner/start", string(raw))
	response := httptest.NewRecorder()
	newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(response, request)
	// Only this positive case may produce synthetic blocks in the test directory.
	if response.Code != http.StatusOK || !r.IsMining() || r.MinerAddress() != address {
		t.Fatalf("authorized fresh start failed: %d", response.Code)
	}
	request = minerControlHTTPV1(http.MethodPost, "/api/miner/stop", "")
	response = httptest.NewRecorder()
	newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatal("authorized fresh miner stop failed")
	}
}

func TestMinerControlUnavailableRuntimeFailsClosed(t *testing.T) {
	readOnlyMuxFixtureV1(t)
	apiReadOnlyMode = false
	previous := nodeRuntime
	t.Cleanup(func() { nodeRuntime = previous })
	address := wallet.NewWallet().GetAddress()
	raw, _ := json.Marshal(minerStartReq{Address: address})
	for _, missing := range []*qnode.Runtime{nil, qnode.NewRuntime(nil, cfg)} {
		nodeRuntime = missing
		for _, path := range []string{"/api/miner/start", "/api/miner/stop", "/api/miner/status"} {
			method := http.MethodPost
			if path == "/api/miner/status" {
				method = http.MethodGet
			}
			response := httptest.NewRecorder()
			newHTTPAPIHandlerV1(nil, NewEndpointLimiter()).ServeHTTP(response, minerControlHTTPV1(method, path, string(raw)))
			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf("unavailable miner runtime status=%d", response.Code)
			}
		}
	}
}

func TestMinerControlOriginMustBeExactRequestOrigin(t *testing.T) {
	for _, origin := range []string{"http://127.0.0.1:8082/", "http://127.0.0.1:8082?", "http://127.0.0.1:8082#", "http://name@127.0.0.1:8082", "https://127.0.0.1:8082"} {
		request := minerControlHTTPV1(http.MethodPost, "/api/miner/start", "")
		request.Header.Set("Origin", origin)
		if minerControlBrowserOriginAllowed(request) {
			t.Fatal("non-origin URL accepted as same-origin")
		}
	}
}

func TestMinerControlUnauthorizedCannotConsumeStopQuota(t *testing.T) {
	r, _, unblock := frozenMinerControlFixtureV1(t)
	handler := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	for range 6 {
		request := minerControlHTTPV1(http.MethodPost, "/api/miner/stop", "")
		request.Header.Set("X-QC-API-KEY", "invalid-fixture-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || !r.IsMining() {
			t.Fatal("unauthorized request changed miner or consumed authenticated quota")
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, minerControlHTTPV1(http.MethodPost, "/api/miner/stop", ""))
	if response.Code != http.StatusOK {
		t.Fatal("unauthorized attempts prevented authorized stop")
	}
	unblock()
}

func TestMinerControlAuthorizedRateLimitRemainsEnabled(t *testing.T) {
	r, address, _ := frozenMinerControlFixtureV1(t)
	handler := newHTTPAPIHandlerV1(nil, NewEndpointLimiter())
	raw, _ := json.Marshal(minerStartReq{Address: address})
	for i := 0; i < 4; i++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, minerControlHTTPV1(http.MethodPost, "/api/miner/start", string(raw)))
		want := http.StatusOK
		if i == 3 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want || !r.IsMining() || r.MinerAddress() != address {
			t.Fatalf("authorized rate limit changed: status=%d want=%d", response.Code, want)
		}
	}
}
