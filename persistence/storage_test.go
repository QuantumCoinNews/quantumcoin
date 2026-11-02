package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageSeparatesMainnetAndTestnet(t *testing.T) {
	root := t.TempDir()
	domain := DefaultPlanV1().Domains[0]

	mainnet, err := ResolveDomainPath(
		StorageConfig{
			RootDir: root,
			Network: NetworkMainnet,
		},
		domain,
	)
	if err != nil {
		t.Fatal(err)
	}

	testnet, err := ResolveDomainPath(
		StorageConfig{
			RootDir: root,
			Network: NetworkTestnet,
		},
		domain,
	)
	if err != nil {
		t.Fatal(err)
	}

	if mainnet == testnet {
		t.Fatal("mainnet and testnet paths must differ")
	}

	if filepath.Dir(mainnet) == filepath.Dir(testnet) {
		t.Fatal("mainnet and testnet must have separate roots")
	}
}

func TestEnsureDomainPathCreatesDirectory(t *testing.T) {
	root := t.TempDir()
	domain := DefaultPlanV1().Domains[0]

	path, err := EnsureDomainPath(
		StorageConfig{
			RootDir: root,
			Network: NetworkTestnet,
		},
		domain,
	)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if !info.IsDir() {
		t.Fatal("resolved persistence path is not a directory")
	}
}

func TestStorageRejectsInvalidNetwork(t *testing.T) {
	_, err := ResolveDomainPath(
		StorageConfig{
			RootDir: t.TempDir(),
			Network: Network("invalid"),
		},
		DefaultPlanV1().Domains[0],
	)
	if err == nil {
		t.Fatal("invalid storage network must fail")
	}
}
