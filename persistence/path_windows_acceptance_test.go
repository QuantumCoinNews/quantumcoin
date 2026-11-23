package persistence

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestPersistenceRegressionJunctionEscape(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	cfg := StorageConfig{RootDir: root, Network: NetworkTestnet}
	domain := DefaultPlanV1().Domains[0]
	networkRoot := filepath.Join(root, string(cfg.Network))
	if err := os.MkdirAll(networkRoot, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(networkRoot, domain.Name)
	cmd := exec.Command("cmd.exe", "/c", "mklink", "/J", link, outside)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("junction fixture unavailable: %v %s", err, output)
	}
	if _, err := EnsureDomainPath(cfg, domain); err == nil {
		t.Error("persistence directory followed junction outside network root")
	}
	if _, err := os.Stat(filepath.Join(outside, domain.SchemaVersion)); !os.IsNotExist(err) {
		t.Error("rejected path created outside directory")
	}
}
