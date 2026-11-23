//go:build !windows

package ai

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

func lockRecordDirectory(dir string) (func(), error) {
	f, err := os.OpenFile(filepath.Join(dir, ".ai-record.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN); _ = f.Close() }, nil
}

func replaceRecord(from, to string) error { return os.Rename(from, to) }
