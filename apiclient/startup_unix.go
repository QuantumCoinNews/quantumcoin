//go:build !windows

package apiclient

import (
	"errors"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func isConnectionRefusedV1(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}

func tryStartupLeaseV1(f *os.File) (func() error, error) {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if errors.Is(err, unix.EWOULDBLOCK) {
		return nil, errStartupLeaseBusyV1
	}
	if err != nil {
		return nil, err
	}
	return func() error { return unix.Flock(int(f.Fd()), unix.LOCK_UN) }, nil
}
