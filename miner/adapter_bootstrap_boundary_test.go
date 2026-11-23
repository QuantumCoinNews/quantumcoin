package miner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

// An explicit mainnet launch is blocked by the canonical runtime policy gate.
// The worker adapter must not bypass that decision by loading a chain itself.
func TestLocalMinerRejectsBlockedRuntimeBeforeCreatingChain(t *testing.T) {
	clearMinerRuntimeDifficultyEnvV1(t)
	t.Setenv("QC_MAINNET", "1")
	directory := t.TempDir()
	cfg := config.Default()
	cfg.PremineLiquidAddress = wallet.NewWallet().GetAddress()
	cfg.PremineLockedAddress = wallet.NewWallet().GetAddress()
	cfg.ChainFile = filepath.Join(directory, config.QCMainnetChainFileV1)
	cfg.BonusFile = filepath.Join(directory, "bonus.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(directory, "config.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	})
	backend, err := NewQCLocalAdapter(QCLocalOpts{ConfigPath: configPath})
	if backend != nil || err == nil || !strings.Contains(err.Error(), "qc local bootstrap:") || !strings.Contains(err.Error(), "explicit mainnet startup blocked") {
		t.Fatalf("expected canonical mainnet gate rejection, backend=%T err=%v", backend, err)
	}
	for _, path := range []string{cfg.ChainFile, cfg.BonusFile} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("blocked bootstrap touched %s: %v", path, err)
		}
	}
}

func TestLocalMinerBootstrapsCanonicalChainInLocalMode(t *testing.T) {
	clearMinerRuntimeDifficultyEnvV1(t)
	t.Setenv("QC_STRICT_CHAIN", "1")
	directory := t.TempDir()
	cfg := config.Default()
	cfg.ChainFile = filepath.Join(directory, config.QCMainnetChainFileV1)
	cfg.BonusFile = filepath.Join(directory, "bonus.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(directory, "config.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	})
	backend, err := NewQCLocalAdapter(QCLocalOpts{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("canonical local bootstrap failed: %v", err)
	}
	local, ok := backend.(*qcLocal)
	if !ok || local.bc == nil || local.cfg == nil {
		t.Fatalf("bootstrap did not supply canonical components: %T", backend)
	}
	if local.cfg.ChainFile != cfg.ChainFile || local.op.ChainPath != cfg.ChainFile {
		t.Fatal("adapter discarded the configured chain path")
	}
	if err := local.bc.ValidateBlockchainSynchronized(); err != nil {
		t.Fatalf("bootstrapped chain failed canonical validation: %v", err)
	}
}
