package persistence

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestStorageLockProcessExitReleasesOwnership(t *testing.T) {
	if path := os.Getenv("QC_STORAGE_LOCK_CHILD"); path != "" {
		lock, err := AcquireFileLock(path, 0)
		if err != nil {
			t.Fatal(err)
		}
		_ = lock // Deliberately exit without Release, simulating abrupt termination.
		os.Exit(0)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	cmd := exec.Command(os.Args[0], "-test.run=^TestStorageLockProcessExitReleasesOwnership$")
	cmd.Env = append(os.Environ(), "QC_STORAGE_LOCK_CHILD="+path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child: %v: %s", err, output)
	}
	lock, err := AcquireFileLock(path, time.Second)
	if err != nil {
		t.Fatalf("crashed owner left storage locked: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}
