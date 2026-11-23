package wallet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/config"
)

func TestMinerSelectionPriorityAndPrivateStoreIsolation(t *testing.T) {
	dir, appDir := t.TempDir(), t.TempDir()
	c := config.Default()
	c.WalletFile = filepath.Join(dir, "store.json")
	o := MinerAddressOptions{Config: c, DataDir: dir, AppDataDir: appDir}
	write := func(path, value string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	app, legacyWallet, legacyMiner, override := NewWallet(), NewWallet(), NewWallet(), NewWallet()
	check := func(want, source string) {
		t.Helper()
		got, err := ResolveMinerAddress(o)
		if err != nil || got.Address != want || got.Source != source {
			t.Fatalf("selection source=%s error=%v", got.Source, err)
		}
	}
	if _, err := ResolveMinerAddress(o); err == nil {
		t.Fatal("missing address accepted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("resolution wrote files")
	}
	write(filepath.Join(appDir, "QuantumCoin", "wallet.json"), `{"address":"`+app.GetAddress()+`"}`)
	check(app.GetAddress(), "legacy-appdata-wallet")
	write(filepath.Join(dir, "wallet_address.txt"), legacyWallet.GetAddress())
	check(legacyWallet.GetAddress(), "wallet_address.txt")
	write(filepath.Join(dir, "miner_address.txt"), legacyMiner.GetAddress())
	check(legacyMiner.GetAddress(), "miner_address.txt")
	write(c.WalletFile, "private key store must not be opened for address selection")
	check(legacyMiner.GetAddress(), "miner_address.txt")
	c.Miner.Address = override.GetAddress()
	c.MinerAddressSource = "QC_MINER"
	check(override.GetAddress(), "QC_MINER")
	o.Explicit = app.GetAddress()
	check(app.GetAddress(), "explicit")
	o.Explicit = "invalid"
	if _, err := ResolveMinerAddress(o); err == nil {
		t.Fatal("invalid explicit address fell back")
	}
	o.Explicit = ""
	c.Miner.Address = "invalid"
	if _, err := ResolveMinerAddress(o); err == nil {
		t.Fatal("invalid configured address fell back")
	}
	c.Miner.Address = ""
	check(legacyMiner.GetAddress(), "miner_address.txt")
}

func TestMinerSelectionRejectsCorruptAndOversizedSources(t *testing.T) {
	for _, kind := range []string{"legacy-empty", "legacy-invalid", "legacy-oversize", "legacy-directory", "app-json"} {
		t.Run(kind, func(t *testing.T) {
			dir, app := t.TempDir(), t.TempDir()
			c := config.Default()
			c.WalletFile = filepath.Join(dir, "store.json")
			o := MinerAddressOptions{Config: c, DataDir: dir, AppDataDir: app}
			target := filepath.Join(dir, "miner_address.txt")
			raw := []byte("invalid")
			switch kind {
			case "legacy-empty":
				raw = nil
			case "legacy-oversize":
				raw = []byte(strings.Repeat("1", 257))
			case "legacy-directory":
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
			case "app-json":
				target = filepath.Join(app, "QuantumCoin", "wallet.json")
				if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
					t.Fatal(err)
				}
				raw = []byte("broken")
			}
			if kind != "legacy-directory" {
				if err := os.WriteFile(target, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := ResolveMinerAddress(o); err == nil {
				t.Fatal("invalid source accepted")
			}
			if kind != "legacy-directory" {
				after, _ := os.ReadFile(target)
				if string(after) != string(raw) {
					t.Fatal("resolver rewrote invalid source")
				}
			}
		})
	}
}
