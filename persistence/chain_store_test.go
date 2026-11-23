package persistence

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyChainStoreRetiredWithoutIO(t *testing.T) {
	p := filepath.Join(t.TempDir(), "missing", "chain.dat")
	if !errors.Is(SaveChainV1(p, nil), ErrLegacyChainStoreRetiredV1) {
		t.Fatal("save must be retired")
	}
	if _, err := LoadChainV1(p); !errors.Is(err, ErrLegacyChainStoreRetiredV1) {
		t.Fatal("load must be retired")
	}
	if _, err := os.Stat(filepath.Dir(p)); !os.IsNotExist(err) {
		t.Fatal("retired writer created a directory")
	}
}
