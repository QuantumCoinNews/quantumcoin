package persistence

import (
	"path/filepath"
	"testing"
	"time"
)

func TestFileLockPreventsConcurrentAcquire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chain.gob")

	first, err := AcquireFileLock(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()

	if _, err := AcquireFileLock(path, 50*time.Millisecond); err == nil {
		t.Fatal("second lock acquisition must fail")
	}
}

func TestFileLockReleaseAllowsReacquire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chain.gob")

	first, err := AcquireFileLock(path, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := first.Release(); err != nil {
		t.Fatal(err)
	}

	second, err := AcquireFileLock(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Release()
}
