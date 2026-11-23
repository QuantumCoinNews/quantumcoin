package node

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestFreshMainnetRejectsMinedTestnetHistoryAndRestartsItsOwnChain(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	t.Setenv("QC_NETWORK", "testnet-local")
	cfg.ChainFile = filepath.Join(t.TempDir(), config.QCTestnetChainFileV1)
	testnet, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	miner := wallet.NewWallet().GetAddress()
	for i := 0; i < 12; i++ {
		if _, err = testnet.MineBlock(miner, config.QCTestnetDefaultDifficultyBitsV1); err != nil {
			t.Fatal(err)
		}
	}
	// Renaming a mined V4 testnet file must not turn it into mainnet history.
	t.Setenv("QC_NETWORK", "mainnet")
	cfg.ChainFile = filepath.Join(t.TempDir(), config.QCMainnetChainFileV1)
	if err = testnet.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	mainnet, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if loaded, err := loadOrCreateBlockchainWithContextV1(cfg, mainnet); err == nil || loaded != nil {
		t.Fatal("mainnet imported mined testnet history")
	}
	after, err := os.ReadFile(cfg.ChainFile)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("mainnet rejection modified historical file", err)
	}
	cfg.ChainFile = filepath.Join(t.TempDir(), config.QCMainnetChainFileV1)
	fresh, err := loadOrCreateBlockchainWithContextV1(cfg, mainnet)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.BestHeightSnapshot() != 0 || fresh.TotalMintedSnapshot() != 0 || len(fresh.BlocksSnapshot()) != 1 {
		t.Fatal("fresh mainnet did not start from genesis only")
	}
	if bytes.Equal(fresh.BlockByIndexSnapshot(0).Hash, testnet.BlockByIndexSnapshot(0).Hash) {
		t.Fatal("testnet and mainnet genesis are identical")
	}
	bits, err := fresh.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fresh.MineBlock(miner, bits); err != nil {
		t.Fatal(err)
	}
	if err = fresh.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	context, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := loadOrCreateBlockchainWithContextV1(cfg, context)
	if err != nil {
		t.Fatal(err)
	}
	if restarted.BestHeightSnapshot() != 1 || !bytes.Equal(restarted.BlockByIndexSnapshot(1).Hash, fresh.BlockByIndexSnapshot(1).Hash) {
		t.Fatal("ordinary mainnet restart lost its own valid history")
	}
}
