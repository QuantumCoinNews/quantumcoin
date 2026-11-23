package blockchain

import (
	"bytes"
	"os"
	"path/filepath"
	"quantumcoin/config"
	"quantumcoin/wallet"
	"strings"
	"testing"
)

func TestNetworkGenesisV2PersistenceAndIsolation(t *testing.T) {
	cfg := config.Default()
	cfg.GenesisProfile = config.NetworkGenesisProfileV2
	testnet, err := NewNetworkBlockchainV2(cfg, config.QCNetworkProfileTestnetLocalV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := testnet.BindRuntimeNetworkIdentityV1("testnet-local"); err != nil {
		t.Fatal(err)
	}
	mainnet, err := NewNetworkBlockchainV2(cfg, config.QCNetworkProfileMainnetV1)
	if err != nil {
		t.Fatal(err)
	}
	if err := mainnet.BindRuntimeNetworkIdentityV1("mainnet"); err != nil {
		t.Fatal(err)
	}
	testID := testnet.NetworkIdentitySnapshotV1()
	mainID := mainnet.NetworkIdentitySnapshotV1()
	testMagic, testBound := testID.DomainMagicV2()
	mainMagic, mainBound := mainID.DomainMagicV2()
	if testID.GenesisHash == mainID.GenesisHash || testMagic == mainMagic || !testBound || !mainBound {
		t.Fatal("network-v2 identities are not independent")
	}
	if testnet.TotalMintedSnapshot() != 0 || mainnet.TotalMintedSnapshot() != 0 {
		t.Fatal("fresh network issued an unallocated genesis subsidy")
	}
	file := filepath.Join(t.TempDir(), "chain.dat")
	if err := testnet.SaveToFile(file); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(file, testnet)
	if err != nil || *loaded.NetworkIdentitySnapshotV1() != *testID {
		t.Fatalf("failed to reload new genesis: %v", err)
	}
	for _, foreign := range []*Blockchain{mainnet, identityTestChain(t, "testnet-local"), identityTestChain(t, "local-release")} {
		if _, err := LoadBlockchainFromFileWithLocalContextV1(file, foreign); err == nil {
			t.Fatal("foreign network loaded new testnet genesis")
		}
	}
}

func identityTestChain(t *testing.T, profile string) *Blockchain {
	t.Helper()
	chain := NewBlockchain(CanonicalInitialMiningRewardV1, int(QCTotalSupplyHardCap))
	if err := chain.BindRuntimeNetworkIdentityV1(profile); err != nil {
		t.Fatal(err)
	}
	return chain
}
func TestStoredNetworkIdentityPreventsLocalMainnetReuse(t *testing.T) {
	file := filepath.Join(t.TempDir(), "chain_data.dat")
	local := identityTestChain(t, "local-release")
	main := identityTestChain(t, "mainnet")
	if err := local.SaveToFile(file); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(file)
	if _, err := LoadBlockchainFromFileWithLocalContextV1(file, main); err == nil {
		t.Fatal("local chain accepted as mainnet")
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(before, after) {
		t.Fatal("failed load changed original file")
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(file, local)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.NetworkIdentitySnapshotV1().Network != "local-release" {
		t.Fatal("network identity lost on restart")
	}
	mainFile := filepath.Join(t.TempDir(), "chain_data.dat")
	if err := main.SaveToFile(file); err == nil {
		t.Fatal("mainnet writer overwrote local destination")
	}
	if err := main.SaveToFile(mainFile); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlockchainFromFileWithLocalContextV1(mainFile, main); err != nil {
		t.Fatal("mainnet cannot reload its own identity", err)
	}
}
func TestMainnetRejectsUntaggedLegacyOrigin(t *testing.T) {
	file := filepath.Join(t.TempDir(), "chain_data.dat")
	legacy := NewBlockchain(CanonicalInitialMiningRewardV1, int(QCTotalSupplyHardCap))
	legacy.SaveToFile(file)
	before, _ := os.ReadFile(file)
	if _, err := LoadBlockchainFromFileWithLocalContextV1(file, identityTestChain(t, "mainnet")); err == nil {
		t.Fatal("untagged legacy accepted as mainnet")
	}
	restored, err := LoadBlockchainFromFileWithLocalContextV1(file, identityTestChain(t, "local-release"))
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(before, after) {
		t.Fatal("legacy adoption wrote the source")
	}
	if restored.NetworkIdentitySnapshotV1().Network != "local-release" {
		t.Fatal("legacy origin not isolated")
	}
}

func TestSameFilenameAndNetworkCannotSubstituteDifferentGenesis(t *testing.T) {
	legacy := identityTestChain(t, "local-release")
	isolated := NewIsolatedLocalBlockchainV1(CanonicalInitialMiningRewardV1, int(QCTotalSupplyHardCap))
	if err := isolated.BindRuntimeNetworkIdentityV1("local-release"); err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct{ source, other *Blockchain }{
		{legacy, isolated}, {isolated, legacy},
	} {
		path := filepath.Join(t.TempDir(), "chain_data.dat")
		if err := pair.source.SaveToFile(path); err != nil {
			t.Fatal(err)
		}
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := LoadBlockchainFromFileWithLocalContextV1(path, pair.source); err != nil {
			t.Fatalf("matching trusted context failed: %v", err)
		}
		if chain, err := LoadBlockchainFromFileWithLocalContextV1(path, pair.other); chain != nil || err == nil ||
			!strings.Contains(err.Error(), "genesis") {
			t.Fatalf("wrong genesis did not fail closed: %v", err)
		}
		if after, err := os.ReadFile(path); err != nil || !bytes.Equal(before, after) {
			t.Fatal("rejected load changed the source file", err)
		}
	}
}
func TestBoundNetworkCannotBeReboundOrSpoofedBeforeSave(t *testing.T) {
	local := identityTestChain(t, "local-release")
	if err := local.BindRuntimeNetworkIdentityV1("mainnet"); err == nil {
		t.Fatal("network rebound")
	}
	local.StoredNetworkIdentityV1.Network = "mainnet"
	if err := local.SaveToFile(filepath.Join(t.TempDir(), "chain_data.dat")); err == nil {
		t.Fatal("public disk marker overrode pinned identity")
	}
}
func TestPeerNetworkIdentityCannotBeBorrowedFromLocalContext(t *testing.T) {
	local := identityTestChain(t, "local-release")
	peer := identityTestChain(t, "mainnet")
	wire := DeserializeBlockchain(SerializeBlockchain(peer))
	if err := wire.BindPeerValidationContextFromLocalSynchronizedV1(local); err == nil {
		t.Fatal("peer network overwritten with local trust")
	}
	wire = DeserializeBlockchain(SerializeBlockchain(local))
	if err := wire.BindPeerValidationContextFromLocalSynchronizedV1(local); err != nil {
		t.Fatal(err)
	}
}
func TestDurableNFTCommitPreservesPinnedNetworkIdentity(t *testing.T) {
	f := nftDurableMiningFixtureV1(t)
	if err := f.chain.BindRuntimeNetworkIdentityV1("local-release"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.chain.MineBlockWithNFTAwardAndPersistSynchronizedV1(wallet.NewWallet().GetAddress(), f.record, f.proof, f.filename); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(f.filename, f.chain)
	if err != nil {
		t.Fatal(err)
	}
	if id := loaded.NetworkIdentitySnapshotV1(); id == nil || id.Network != "local-release" {
		t.Fatal("candidate disk commit discarded network identity")
	}
}

func TestConcurrentNetworksCannotOverwriteOneDestination(t *testing.T) {
	file := filepath.Join(t.TempDir(), "chain_data.dat")
	local, main := identityTestChain(t, "local-release"), identityTestChain(t, "mainnet")
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, chain := range []*Blockchain{local, main} {
		go func(chain *Blockchain) { <-start; results <- chain.SaveToFile(file) }(chain)
	}
	close(start)
	successes := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("different networks installed the same path: successes=%d", successes)
	}
	saved, err := LoadBlockchainFromFile(file)
	if err != nil {
		t.Fatal(err)
	}
	trusted := local
	if saved.StoredNetworkIdentityV1.Network == "mainnet" {
		trusted = main
	}
	if _, err := LoadBlockchainFromFileWithLocalContextV1(file, trusted); err != nil {
		t.Fatal("winner file corrupted", err)
	}
}
func TestUnboundWriterCannotOverwriteTaggedDestination(t *testing.T) {
	file := filepath.Join(t.TempDir(), "chain_data.dat")
	tagged := identityTestChain(t, "local-release")
	if err := tagged.SaveToFile(file); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(file)
	raw := NewBlockchain(CanonicalInitialMiningRewardV1, int(QCTotalSupplyHardCap))
	if err := raw.SaveToFile(file); err == nil {
		t.Fatal("untagged writer erased pinned network")
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(before, after) {
		t.Fatal("rejected writer changed destination")
	}
}
