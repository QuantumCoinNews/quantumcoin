package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestNativeGUIWorkerAuthorizationFailsBeforeHashing(t *testing.T) {
	previous := config.Current()
	t.Cleanup(func() { config.Set(previous) })
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("QC_DATA_DIR", dir)
	t.Setenv("QC_NETWORK", "testnet-local")
	t.Setenv("QC_NET_MAGIC", config.QCTestnetMagicV1)
	t.Setenv("QC_HTTP_PORT", ":18082")
	t.Setenv("HTTP_PORT", "")
	t.Setenv("QC_CHAIN_FILE", filepath.Join(dir, config.QCTestnetChainFileV1))
	t.Setenv("QC_WALLET_FILE", filepath.Join(dir, "unused-wallet.json"))
	t.Setenv("QC_DEFAULT_DIFFICULTY_BITS", "8")
	address := wallet.NewWallet().GetAddress()
	t.Setenv("ADDR", address)
	c := config.Default()
	c.GenesisProfile = config.NetworkGenesisProfileV4
	c.DefaultDifficultyBits = config.QCTestnetDefaultDifficultyBitsV1
	c.HTTPPort = ":18082"
	c.P2PPort = ":13001"
	c.ChainFile = filepath.Join(dir, config.QCTestnetChainFileV1)
	c.WalletFile = filepath.Join(dir, "unused-wallet.json")
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.json")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("QC_CONFIG_FILE", path)
	for _, mode := range []string{"invalid-token", "forbidden", "empty-token", "invalid-address"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != "/api/mine/job" || r.Method != http.MethodGet || r.URL.Query().Get("address") != address {
					t.Error("GUI used a legacy control route or changed reward address")
				}
				if r.Header.Get("X-QC-API-KEY") != "gui-miner-fixture-token" || r.URL.Query().Has("token") {
					t.Error("GUI did not transport credentials through the header only")
				}
				status := http.StatusUnauthorized
				if mode == "forbidden" {
					status = http.StatusForbidden
				}
				http.Error(w, "denied", status)
			}))
			defer server.Close()
			t.Setenv("QC_API_BASE", server.URL)
			t.Setenv("QC_API_TOKEN", "gui-miner-fixture-token")
			if mode == "empty-token" {
				t.Setenv("QC_API_TOKEN", "")
			}
			if mode == "invalid-address" {
				t.Setenv("ADDR", "invalid-address")
			}
			if err := startNativeMining(); err == nil {
				_ = stopNativeMining()
				t.Fatal("denied GUI miner start reported success")
			}
			if nativeMinerAlive() {
				_ = stopNativeMining()
				t.Fatal("unauthorized GUI worker started hashing")
			}
			want := int32(1)
			if mode == "empty-token" || mode == "invalid-address" {
				want = 0
			}
			if calls.Load() != want {
				t.Fatalf("unexpected preflight count=%d want=%d", calls.Load(), want)
			}
		})
	}
	if _, err = os.Stat(c.ChainFile); !os.IsNotExist(err) {
		t.Fatal("failed GUI mining created a chain file")
	}
	if _, err = os.Stat(c.WalletFile); !os.IsNotExist(err) {
		t.Fatal("failed GUI mining created a wallet file")
	}
}
