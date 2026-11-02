package blockchain

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBlockchainRejectsUnauthorizedNFTRecordV1(t *testing.T) {
	chain := NewBlockchain(1, 0)
	directory := t.TempDir()
	cleanPath := filepath.Join(directory, "clean-chain.dat")

	if err := chain.SaveToFile(cleanPath); err != nil {
		t.Fatalf("save clean baseline: %v", err)
	}

	clean, err := LoadBlockchainFromFile(cleanPath)
	if err != nil {
		t.Fatalf("load clean baseline: %v", err)
	}
	if clean == nil || len(clean.Blocks) != len(chain.Blocks) {
		t.Fatal("clean baseline did not preserve block count")
	}
	if !bytes.Equal(clean.Blocks[0].Hash, chain.Blocks[0].Hash) {
		t.Fatal("clean baseline changed genesis hash")
	}
	if !clean.IsValidChain() {
		t.Fatal("clean baseline is invalid after loading")
	}

	chain.Blocks[0].NFTMintRecordsV1 = []NFTMintRecordV1{
		nftRecordSetFixtureV1(1, 0x55),
	}

	if chain.IsValidChain() {
		t.Fatal("unauthorized NFT fixture unexpectedly passed chain validation")
	}

	poisonedPath := filepath.Join(directory, "unauthorized-nft-chain.dat")
	if err := os.WriteFile(
		poisonedPath,
		SerializeBlockchain(chain),
		0o600,
	); err != nil {
		t.Fatalf("write unauthorized NFT fixture: %v", err)
	}

	loaded, err := LoadBlockchainFromFile(poisonedPath)
	if err == nil {
		t.Fatal("QMA0068_PERSISTENCE_RED: loader accepted a chain containing an unauthorized NFT record")
	}
	if loaded != nil {
		t.Fatal("rejected NFT file returned a usable blockchain")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "nft") {
		t.Fatalf("load failed outside NFT validation: %v", err)
	}
}
