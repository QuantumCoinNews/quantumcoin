package node

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
)

func TestMainnetRequiresFreshV4AndPreservesExistingLocalData(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	t.Setenv("QC_NETWORK", "mainnet")
	cfg.ChainFile = filepath.Join(t.TempDir(), config.QCMainnetChainFileV1)
	for _, profile := range []string{"", config.NetworkGenesisProfileV2, config.NetworkGenesisProfileV3} {
		cfg.GenesisProfile = profile
		if _, err := newRuntimeChainContextV1(cfg); err == nil || !strings.Contains(err.Error(), "starts fresh") {
			t.Fatal("mainnet accepted old genesis profile", err)
		}
	}
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	context, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	local := blockchain.NewIsolatedLocalBlockchainV1(50, int(blockchain.QCTotalSupplyHardCap))
	if err = local.BindRuntimeNetworkIdentityV1("local-release"); err != nil {
		t.Fatal(err)
	}
	if err = local.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	before := local.BlocksSnapshot()
	if chain, err := loadOrCreateBlockchainWithContextV1(cfg, context); err == nil || chain != nil {
		t.Fatal("mainnet imported existing local history")
	}
	after, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(cfg.ChainFile, local)
	if err != nil || !bytes.Equal(before[0].Hash, after.BlockByIndexSnapshot(0).Hash) {
		t.Fatal("rejected mainnet start damaged local history", err)
	}
}

func TestFreshTestnetAndMainnetStartWithIndependentGenesisAndNoImportedHistory(t *testing.T) {
	var genesis [][]byte
	for _, network := range []string{"testnet-local", "mainnet"} {
		t.Run(network, func(t *testing.T) {
			cfg := defaultRuntimePolicyConfigV1(t)
			t.Setenv("QC_NETWORK", network)
			cfg.GenesisProfile = config.NetworkGenesisProfileV4
			filename := config.QCTestnetChainFileV1
			if network == "mainnet" {
				filename = config.QCMainnetChainFileV1
			}
			cfg.ChainFile = filepath.Join(t.TempDir(), filename)
			context, err := newRuntimeChainContextV1(cfg)
			if err != nil {
				t.Fatal(err)
			}
			chain, err := loadOrCreateBlockchainWithContextV1(cfg, context)
			if err != nil {
				t.Fatal(err)
			}
			if len(chain.BlocksSnapshot()) != 1 || chain.TotalMintedSnapshot() != 0 {
				t.Fatal("fresh test fixture imported historical blocks or balances")
			}
			genesis = append(genesis, chain.BlockByIndexSnapshot(0).Hash)
		})
	}
	if bytes.Equal(genesis[0], genesis[1]) {
		t.Fatal("testnet and mainnet share a genesis")
	}
}
