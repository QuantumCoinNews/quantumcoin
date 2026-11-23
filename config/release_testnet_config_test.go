package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseTestnetConfigUsesCanonicalLoader(t *testing.T) {
	previous := Current()
	t.Cleanup(func() { Set(previous) })
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(strings.ToUpper(key), "QC_") || strings.EqualFold(key, "HTTP_PORT") {
			t.Setenv(key, "")
		}
	}
	t.Setenv("QC_TESTNET", "1")
	t.Setenv("QC_NET_MAGIC", QCTestnetMagicV1)
	data := t.TempDir()
	cfg, err := LoadWithDataDir(filepath.Join("..", "release", "config.testnet.json"), data)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateRuntimeNetworkProfileV1(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.GenesisProfile != NetworkGenesisProfileV4 {
		t.Fatal("fresh testnet release profile lacks genesis-bound signatures")
	}
	if cfg.ChainFile != filepath.Join(data, QCTestnetChainFileV1) || cfg.HTTPPort != ":18082" || cfg.P2PPort != ":13001" {
		t.Fatalf("wrong testnet storage or ports: %s %s %s", cfg.ChainFile, cfg.HTTPPort, cfg.P2PPort)
	}
}
