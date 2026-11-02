package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"quantumcoin/blockchain"
)

func TestChainStoreRoundTrip(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "mainnet", "chain.gob")

	chain := blockchain.NewBlockchain(1, 0)
	blocks := chain.BlocksSnapshot()

	if err := SaveChainV1(path, blocks); err != nil {
		t.Fatalf("save chain: %v", err)
	}

	loaded, err := LoadChainV1(path)
	if err != nil {
		t.Fatalf("load chain: %v", err)
	}

	if len(loaded) != len(blocks) {
		t.Fatalf("loaded block count = %d, want %d", len(loaded), len(blocks))
	}

	if loaded[0].Index != blocks[0].Index {
		t.Fatal("loaded genesis index mismatch")
	}

	if loaded[0].Hash == nil {
		t.Fatal("loaded genesis hash is nil")
	}
}

func TestChainStoreRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chain.gob")

	if err := os.WriteFile(path, []byte("corrupt-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadChainV1(path); err == nil {
		t.Fatal("corrupt chain file must fail")
	}
}

func TestChainStoreRejectsEmptyChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chain.gob")

	if err := SaveChainV1(path, nil); err == nil {
		t.Fatal("empty chain must fail")
	}
}

func TestChainStoreDoesNotLeaveTemporaryFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "chain.gob")

	chain := blockchain.NewBlockchain(1, 0)
	if err := SaveChainV1(path, chain.BlocksSnapshot()); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temporary chain file remains after save")
	}
}
