package node

import (
	"os"
	"path/filepath"
	"quantumcoin/config"
	"strings"
	"testing"
)

func TestRuntimeEnvironmentRejectsBeforeChainCreationV1(t *testing.T) {
	for key, value := range map[string]string{"QC_MINING_DELAY_MS": "test-only-invalid", "qc_mining_delay_ms": "test-only-invalid", "QC_API_LISTEN": "127.0.0.1:9999"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, value)
			c := config.Default()
			c.ChainFile = filepath.Join(t.TempDir(), "must-not-exist.dat")
			_, err := BootstrapWithConfig(c, false)
			if err == nil || !strings.Contains(strings.ToUpper(err.Error()), strings.ToUpper(key)) {
				t.Fatal("invalid runtime config not rejected")
			}
			if _, err := os.Stat(c.ChainFile); !os.IsNotExist(err) {
				t.Fatal("rejected startup touched chain file")
			}
		})
	}
}
