package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithDataDirBindsPathsBeforePublishing(t *testing.T) {
	clearConfigLoaderEnvV1(t)
	previous := Current()
	t.Cleanup(func() { Set(previous) })
	file := writeConfigFixtureV1(t, `{"chain_file":"ledger.dat","wallet_file":"keys/store.json","bonus_file":"bonus.json"}`)
	base := filepath.Join(t.TempDir(), "data-not-created")
	absoluteWallet := filepath.Join(t.TempDir(), "external-wallet.json")
	t.Setenv("QC_WALLET_FILE", absoluteWallet)
	t.Setenv("QC_CHAIN_FILE", "env-ledger.dat")
	before, _ := os.Getwd()
	c, err := LoadWithDataDir(file, base)
	if err != nil {
		t.Fatal(err)
	}
	if c.ChainFile != filepath.Join(base, "env-ledger.dat") || c.WalletFile != absoluteWallet || c.BonusFile != filepath.Join(base, "bonus.json") {
		t.Fatal("storage paths/precedence wrong")
	}
	if Current().WalletFile != absoluteWallet || Current().ChainFile != c.ChainFile {
		t.Fatal("wallet/global config disagrees with runtime config")
	}
	after, _ := os.Getwd()
	if before != after {
		t.Fatal("config changed cwd")
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatal("config loader created directories")
	}
	t.Setenv("QC_TESTNET", "1")
	if _, err := LoadWithDataDir(file, base); err == nil {
		t.Fatal("path anchoring bypassed network separation")
	}
}
