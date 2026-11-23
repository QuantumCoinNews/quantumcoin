//go:build windows

package apiclient

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func isConnectionRefusedV1(err error) bool {
	return errors.Is(err, windows.WSAECONNREFUSED)
}

func tryStartupLeaseV1(f *os.File) (func() error, error) {
	var overlap windows.Overlapped
	err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlap)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return nil, errStartupLeaseBusyV1
	}
	if err != nil {
		return nil, err
	}
	return func() error { return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &overlap) }, nil
}
