package node

import (
	"bytes"
	"encoding/gob"
	"os"
	"path/filepath"
	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consensus"
	"quantumcoin/wallet"
	"strings"
	"testing"
)

func TestRuntimeSelectsNetworkV2OnlyByExplicitProfile(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.ChainFile = filepath.Join(filepath.Dir(cfg.ChainFile), config.QCTestnetChainFileV1)
	cfg.PremineLiquidAddress = wallet.NewWallet().GetAddress()
	cfg.PremineLockedAddress = wallet.NewWallet().GetAddress()
	t.Setenv("QC_NETWORK", "testnet-local")
	legacy, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.GenesisProfile = config.NetworkGenesisProfileV2
	testnet, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("QC_NETWORK", "mainnet")
	cfg.ChainFile = filepath.Join(filepath.Dir(cfg.ChainFile), config.QCMainnetChainFileV1)
	// Preserve historical V2 genesis evidence without publishing an old-profile
	// mainnet runtime: the new mainnet startup explicitly requires V4.
	mainnet, err := blockchain.NewNetworkBlockchainV2(cfg, config.QCNetworkProfileMainnetV1)
	if err != nil {
		t.Fatal(err)
	}
	if err = mainnet.BindRuntimeNetworkIdentityV1("mainnet"); err != nil {
		t.Fatal(err)
	}
	if legacy.NetworkIdentitySnapshotV1().GenesisDomain != "" ||
		testnet.NetworkIdentitySnapshotV1().GenesisDomain != blockchain.TestnetGenesisDomainV2 ||
		mainnet.NetworkIdentitySnapshotV1().GenesisDomain != blockchain.MainnetGenesisDomainV2 {
		t.Fatal("unexpected runtime genesis selection")
	}
	if len(testnet.Blocks[0].Transactions) != 2 || len(mainnet.Blocks[0].Transactions) != 2 ||
		len(testnet.Blocks[0].Transactions[1].Outputs) != 2 ||
		bytes.Equal(testnet.Blocks[0].Transactions[1].ID, mainnet.Blocks[0].Transactions[1].ID) ||
		testnet.TotalMintedSnapshot() != 1_275_000 {
		t.Fatal("network-v2 founder outpoints or issuance are incorrect")
	}
	path := filepath.Join(t.TempDir(), "network-v2.dat")
	if err := testnet.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	if _, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(path, testnet); err != nil {
		t.Fatal("network-v2 founder allocations did not reload", err)
	}
}

func TestIsolatedLocalGenesisRejectsMainnetEvenAfterRelabeling(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	dir := filepath.Dir(cfg.ChainFile)
	cfg.ChainFile = filepath.Join(dir, config.QCIsolatedLocalChainFileV1)
	cfg.HTTPPort = config.QCIsolatedLocalHTTPPortV1
	cfg.P2PPort = config.QCIsolatedLocalP2PPortV1
	cfg.PremineLiquidAddress = wallet.NewWallet().GetAddress()
	cfg.PremineLockedAddress = wallet.NewWallet().GetAddress()
	previous := config.Current()
	t.Cleanup(func() { config.Set(previous) })
	if err := config.SetValidated(cfg); err != nil {
		t.Fatal(err)
	}
	runtime, err := BootstrapWithConfig(cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	local, _ := runtime.Snapshot()
	main := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	if err := main.BindRuntimeNetworkIdentityV1("mainnet"); err != nil {
		t.Fatal(err)
	}
	id := local.NetworkIdentitySnapshotV1()
	if id.GenesisDomain != blockchain.IsolatedLocalGenesisDomainV1 || id.GenesisHash == main.NetworkIdentitySnapshotV1().GenesisHash {
		t.Fatal("isolated local genesis is not a separate PoW domain")
	}
	if len(local.Blocks[0].Transactions) < 2 || len(main.Blocks[0].Transactions) < 2 ||
		bytes.Equal(local.Blocks[0].Transactions[1].ID, main.Blocks[0].Transactions[1].ID) {
		t.Fatal("isolated prefund shares a legacy outpoint")
	}
	if err := local.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	if _, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(cfg.ChainFile, local); err != nil {
		t.Fatal("local chain cannot reopen", err)
	}

	// An attacker can edit the disk tag, but the canonical genesis remains.
	local.StoredNetworkIdentityV1 = main.NetworkIdentitySnapshotV1()
	forgedPath := filepath.Join(dir, config.QCMainnetChainFileV1)
	file, err := os.Create(forgedPath)
	if err != nil {
		t.Fatal(err)
	}
	err = gob.NewEncoder(file).Encode(local)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(forgedPath, main); err == nil || !strings.Contains(err.Error(), "genesis differs") {
		t.Fatalf("retagged local chain was not rejected at the genesis boundary: %v", err)
	}
}

func TestStartupLoaderRefusesLocalDiskWhenModeChangesToMainnet(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	previous := config.Current()
	t.Cleanup(func() { config.Set(previous) })
	cfg.PremineLiquidAddress = wallet.NewWallet().GetAddress()
	cfg.PremineLockedAddress = wallet.NewWallet().GetAddress()
	if err := config.SetValidated(cfg); err != nil {
		t.Fatal(err)
	}
	local, err := BootstrapWithConfig(cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := local.SaveChain(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(cfg.ChainFile)
	t.Setenv("QC_MAINNET", "1")
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	// Explicit mainnet remains blocked by the final-fee preflight gate.
	// Exercise its canonical disk-loading stage independently, without weakening that gate.
	if chain, _, err := loadBlockchainWithNFTStartupV1(cfg, consensus.ActivationPolicyV1{}, nil, ""); err == nil || chain != nil {
		t.Fatal("local file became an explicit mainnet runtime")
	} else if !strings.Contains(err.Error(), "network identity") {
		t.Fatal("startup failed before exercising the network identity guard", err)
	}
	after, _ := os.ReadFile(cfg.ChainFile)
	if !bytes.Equal(before, after) {
		t.Fatal("rejected startup changed original chain")
	}
}

func TestBootstrapRejectsUnsupportedRewardRolesBeforePublishingRuntime(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.RewardPctMiner = 70
	cfg.RewardPctStake = 10
	cfg.RewardPctDev = 10
	cfg.RewardPctBurn = 10
	runtime, err := BootstrapWithConfig(cfg, true)
	if err == nil || runtime != nil {
		t.Fatal("unsupported reward split reached runtime")
	}
	if _, err := os.Stat(cfg.ChainFile); !os.IsNotExist(err) {
		t.Fatal("invalid reward config wrote chain state", err)
	}
}
