package node

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
)

func TestNetworkV4StartupPreservesLegacyFileAndRejectsSilentMigration(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	t.Setenv("QC_NETWORK", "testnet-local")
	cfg.ChainFile = filepath.Join(t.TempDir(), config.QCTestnetChainFileV1)
	cfg.GenesisProfile = config.NetworkGenesisProfileV3
	old, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = old.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(cfg.ChainFile)
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	fresh, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.NetworkIdentitySnapshotV1().GenesisDomain != blockchain.TestnetGenesisDomainV4 {
		t.Fatal("startup selected wrong genesis")
	}
	if _, err = loadOrCreateBlockchainWithContextV1(cfg, fresh); err == nil {
		t.Fatal("old chain silently migrated to new signatures")
	}
	after, _ := os.ReadFile(cfg.ChainFile)
	if !bytes.Equal(before, after) {
		t.Fatal("failed migration modified old data")
	}
	cfg.GenesisProfile = config.NetworkGenesisProfileV3
	if _, err = loadOrCreateBlockchainWithContextV1(cfg, old); err != nil {
		t.Fatal("old profile can no longer reload its history", err)
	}
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	cfg.ChainFile = filepath.Join(t.TempDir(), config.QCTestnetChainFileV1)
	if created, err := loadOrCreateBlockchainWithContextV1(cfg, fresh); err != nil || created.NetworkIdentitySnapshotV1().GenesisDomain != blockchain.TestnetGenesisDomainV4 {
		t.Fatal("fresh V4 startup failed", err)
	}
}

func TestPublicTestnetRequiresGenesisBoundTransactionSignatures(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	t.Setenv("QC_NETWORK", "testnet-public")
	cfg.ChainFile = filepath.Join(t.TempDir(), config.QCTestnetChainFileV1)
	for _, profile := range []string{"", config.NetworkGenesisProfileV2, config.NetworkGenesisProfileV3} {
		cfg.GenesisProfile = profile
		if _, err := newRuntimeChainContextV1(cfg); err == nil || !strings.Contains(err.Error(), "network-v4") {
			t.Fatal("public testnet accepted a legacy signing profile", err)
		}
	}
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	if _, err := newRuntimeChainContextV1(cfg); err != nil {
		t.Fatal(err)
	}
}
