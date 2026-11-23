package persistence

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
)

func tryStorageLock(f *os.File) (func() error, error) {
	var ov windows.Overlapped
	h := windows.Handle(f.Fd())
	err := windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ov)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return nil, ErrStorageLocked
	}
	if err != nil {
		return nil, err
	}
	return func() error { return windows.UnlockFileEx(h, 0, 1, 0, &ov) }, nil
}
