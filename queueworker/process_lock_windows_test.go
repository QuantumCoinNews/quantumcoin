package queueworker

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestQueueStoreCrashReleasesProcessLock(t *testing.T) {
	if path := os.Getenv("QC_QUEUE_LOCK_CHILD"); path != "" {
		release, err := acquireProcessLockV1(path)
		if err != nil {
			t.Fatal(err)
		}
		_ = release
		os.Exit(0)
	}
	path := filepath.Join(t.TempDir(), "queue.lock")
	child := exec.Command(os.Args[0], "-test.run=^TestQueueStoreCrashReleasesProcessLock$")
	child.Env = append(os.Environ(), "QC_QUEUE_LOCK_CHILD="+path)
	child.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("child: %v: %s", err, output)
	}
	release, err := acquireProcessLockV1(path)
	if err != nil {
		t.Fatalf("crashed process retained ownership: %v", err)
	}
	release()
}
