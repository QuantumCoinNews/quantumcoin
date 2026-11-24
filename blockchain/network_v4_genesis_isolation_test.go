package blockchain

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"quantumcoin/config"
)

func TestNetworkV4FreshPublicGenesisIsolationAndRestart(t *testing.T) {
	cfg := config.Default()
	cfg.GenesisProfile = config.NetworkGenesisProfileV4

	testnet, err := NewNetworkBlockchainV4(cfg, config.QCNetworkProfileTestnetPublicV1)
	if err != nil {
		t.Fatal(err)
	}
	mainnet, err := NewNetworkBlockchainV4(cfg, config.QCNetworkProfileMainnetV1)
	if err != nil {
		t.Fatal(err)
	}
	testnetReplica, err := NewNetworkBlockchainV4(cfg, config.QCNetworkProfileTestnetPublicV1)
	if err != nil {
		t.Fatal(err)
	}
	mainnetReplica, err := NewNetworkBlockchainV4(cfg, config.QCNetworkProfileMainnetV1)
	if err != nil {
		t.Fatal(err)
	}

	testnetIdentity := testnet.NetworkIdentitySnapshotV1()
	mainnetIdentity := mainnet.NetworkIdentitySnapshotV1()
	testnetReplicaIdentity := testnetReplica.NetworkIdentitySnapshotV1()
	mainnetReplicaIdentity := mainnetReplica.NetworkIdentitySnapshotV1()
	if testnetIdentity == nil || mainnetIdentity == nil ||
		testnetReplicaIdentity == nil || mainnetReplicaIdentity == nil {
		t.Fatal("fresh V4 network identity is missing")
	}
	if *testnetReplicaIdentity != *testnetIdentity || *mainnetReplicaIdentity != *mainnetIdentity {
		t.Fatal("same V4 network profile did not reproduce its genesis identity")
	}
	if testnetIdentity.Network != "testnet" || mainnetIdentity.Network != "mainnet" ||
		testnetIdentity.GenesisHash == mainnetIdentity.GenesisHash {
		t.Fatal("fresh V4 public networks do not have distinct pinned identities")
	}

	for name, chain := range map[string]*Blockchain{"testnet": testnet, "mainnet": mainnet} {
		if chain.BestHeightSnapshot() != 0 || chain.TotalMintedSnapshot() != 0 {
			t.Fatalf("fresh %s must start at genesis with zero issuance", name)
		}
		genesis := chain.BlockByIndexSnapshot(0)
		if genesis == nil || len(genesis.Transactions) != 1 ||
			!isNetworkGenesisMarkerV1(genesis.Transactions[0]) {
			t.Fatalf("fresh %s genesis is missing its zero-value network marker", name)
		}
	}

	testnetFile := filepath.Join(t.TempDir(), "chain_data.dat")
	if err := testnet.SaveToFile(testnetFile); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(testnetFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBlockchainFromFileWithLocalContextV1(testnetFile, mainnet); err == nil {
		t.Fatal("mainnet accepted a testnet genesis file")
	}
	after, err := os.ReadFile(testnetFile)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("rejected cross-network load changed the testnet file", err)
	}
	reloadedTestnet, err := LoadBlockchainFromFileWithLocalContextV1(testnetFile, testnet)
	if err != nil || reloadedTestnet.BestHeightSnapshot() != 0 ||
		reloadedTestnet.NetworkIdentitySnapshotV1().GenesisHash != testnetIdentity.GenesisHash {
		t.Fatalf("testnet genesis did not reload intact: %v", err)
	}

	mainnetFile := filepath.Join(t.TempDir(), "chain_data.dat")
	if err := mainnet.SaveToFile(mainnetFile); err != nil {
		t.Fatal(err)
	}
	reloadedMainnet, err := LoadBlockchainFromFileWithLocalContextV1(mainnetFile, mainnet)
	if err != nil || reloadedMainnet.BestHeightSnapshot() != 0 ||
		reloadedMainnet.NetworkIdentitySnapshotV1().GenesisHash != mainnetIdentity.GenesisHash {
		t.Fatalf("mainnet did not restart from its own fresh genesis: %v", err)
	}
}
