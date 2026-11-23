package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"quantumcoin/config"
	"quantumcoin/utils"
	"strings"
	"testing"

	"quantumcoin/wallet"
)

func TestOfflineCLIChild(t *testing.T) {
	if os.Getenv("QC_OFFLINE_TEST_CHILD") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("QC_OFFLINE_TEST_ARGS")), &args); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Getwd()
	os.Args = append([]string{os.Args[0]}, args...)
	runNodeCLI()
	after, _ := os.Getwd()
	if before != after || nodeRuntime != nil {
		t.Fatal("offline dispatch changed cwd or bootstrapped node")
	}
}

func offlineCLIChildV1(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(args)
	cmd := exec.Command(exe, "-test.run=^TestOfflineCLIChild$")
	cmd.Dir = cwd
	for _, e := range os.Environ() {
		name, _, _ := strings.Cut(e, "=")
		switch strings.ToUpper(name) {
		case "QC_CONFIG_FILE", "QC_DATA_DIR", "QC_WALLET_FILE":
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	cmd.Env = append(cmd.Env, "QC_OFFLINE_TEST_CHILD=1", "QC_OFFLINE_TEST_ARGS="+string(raw))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("offline command failed: %v", err)
	}
	return string(out)
}

func TestOfflineCommandsIgnoreBrokenNodeConfigAndChain(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"not-a-command"}, {"send"}, {"sign-burn", "x", "1", "bad-option"}, {"--data-dir"}, {"--unknown"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"config.json", "chain_data.dat", "wallet_data.json"} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte("broken sentinel"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			out := offlineCLIChildV1(t, dir, args...)
			if !strings.Contains(out, "Usage:") || strings.Contains(out, "bootstrap failed") {
				t.Fatal("help/invalid command did not bypass bootstrap")
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 3 {
				t.Fatal("offline help wrote new files")
			}
			for _, name := range []string{"config.json", "chain_data.dat", "wallet_data.json"} {
				got, _ := os.ReadFile(filepath.Join(dir, name))
				if string(got) != "broken sentinel" {
					t.Fatal("offline command rewrote files")
				}
			}
		})
	}
}

func TestOfflineWalletPersistsRecoverableKeysAtExplicitLocation(t *testing.T) {
	for _, command := range []string{"newaddr", "newaddr-priv"} {
		t.Run(command, func(t *testing.T) {
			dir, data := t.TempDir(), t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("invalid JSON"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "chain_data.dat"), []byte("broken chain"), 0600); err != nil {
				t.Fatal(err)
			}
			out := offlineCLIChildV1(t, dir, "--data-dir", data, "--wallet-file", "nested/store.json", command)
			raw, err := os.ReadFile(filepath.Join(data, "nested", "store.json"))
			if err != nil {
				t.Fatal("wallet was not persisted at requested path")
			}
			var store struct {
				Wallets map[string]string `json:"wallets"`
				Default string            `json:"default"`
			}
			if json.Unmarshal(raw, &store) != nil || len(store.Wallets) != 1 {
				t.Fatal("invalid saved wallet")
			}
			key := store.Wallets[store.Default]
			priv, err := wallet.ImportPrivateKeyHex(key)
			if err != nil {
				t.Fatal("saved key cannot be restored")
			}
			public, err := utils.SecpPublicBytesV1(&priv.PublicKey)
			if err != nil {
				t.Fatal(err)
			}
			w := &wallet.Wallet{PrivateKey: priv, PublicKey: public}
			if w.GetAddress() != store.Default || !strings.Contains(out, store.Default) {
				t.Fatal("reported wallet differs from persisted key")
			}
			if strings.Contains(out, key) != (command == "newaddr-priv") {
				t.Fatal("private key output contract violated")
			}
			for _, name := range []string{"chain_data.dat", "wallet_data.json", "config.json"} {
				if _, err := os.Stat(filepath.Join(data, name)); !os.IsNotExist(err) {
					t.Fatal("wallet command created node files")
				}
			}
			got, _ := os.ReadFile(filepath.Join(dir, "chain_data.dat"))
			if !bytes.Equal(got, []byte("broken chain")) {
				t.Fatal("wallet command changed chain")
			}
		})
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wallet_data.json"), []byte("corrupt store"), 0600); err != nil {
		t.Fatal(err)
	}
	out := offlineCLIChildV1(t, dir, "newaddr")
	if strings.Contains(out, "New Wallet Address:") || !strings.Contains(out, "wallet persistence failed") {
		t.Fatal("wallet failure reported success")
	}
	got, _ := os.ReadFile(filepath.Join(dir, "wallet_data.json"))
	if string(got) != "corrupt store" {
		t.Fatal("corrupt wallet overwritten")
	}
}

func TestCLIPathsPrecedenceAndNoFilesystemMutation(t *testing.T) {
	t.Setenv("QC_CONFIG_FILE", "env-config.json")
	t.Setenv("QC_DATA_DIR", filepath.Join(t.TempDir(), "env-data"))
	t.Setenv("QC_WALLET_FILE", "env-wallet.json")
	base := filepath.Join(t.TempDir(), "not-created")
	p, err := parseCLIPathsV1([]string{"--data-dir", base, "--config=flag-config.json", "--wallet-file=flag-wallet.json", "newaddr"})
	if err != nil {
		t.Fatal(err)
	}
	wantConfig, _ := filepath.Abs("flag-config.json")
	if p.DataDir != base || p.Config != wantConfig || p.Wallet != filepath.Join(base, "flag-wallet.json") {
		t.Fatal("flag precedence/anchoring wrong")
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatal("path resolution created directory")
	}
	t.Setenv("QC_CONFIG_FILE", "")
	t.Setenv("QC_DATA_DIR", "")
	t.Setenv("QC_WALLET_FILE", "")
	p, err = parseCLIPathsV1(nil)
	if err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	if p.DataDir != cwd || p.Config != filepath.Join(cwd, "config.json") || p.Wallet != filepath.Join(cwd, "wallet_data.json") {
		t.Fatal("defaults are not caller-relative")
	}
}

func TestMinerAddressLookupUsesSelectedCLIPaths(t *testing.T) {
	t.Setenv("APPDATA", "")
	t.Setenv("QC_MINER", "")
	old := activeCLIPathsV1
	oldCfg := cfg
	t.Cleanup(func() { activeCLIPathsV1 = old; cfg = oldCfg })
	dir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "chosen-config.json")
	activeCLIPathsV1 = cliPathsV1{DataDir: dir, Config: configPath}
	address := wallet.NewWallet().GetAddress()
	cfg = config.Default()
	cfg.Miner.Address = address
	cfg.WalletFile = filepath.Join(dir, "wallet_data.json")
	if err := os.WriteFile(configPath, []byte("invalid changed config must not be reread"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := lookupDefaultAddressV1(); got != address {
		t.Fatal("selected config ignored")
	}
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
	cfg.Miner.Address = ""
	if err := os.WriteFile(filepath.Join(dir, "miner_address.txt"), []byte(address), 0600); err != nil {
		t.Fatal(err)
	}
	if got := lookupDefaultAddressV1(); got != address {
		t.Fatal("selected data directory ignored")
	}
}
