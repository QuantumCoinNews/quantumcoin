//go:build unix

package blockchain

import (
	"golang.org/x/sys/unix"
	"os"
)

func lockChainFileOSV1(filename, _ string) (func(), error) {
	// The lock inode remains stable across destination renames. Do not unlink
	// it on release: other processes may already be waiting on this inode.
	file, err := os.OpenFile(filename+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN); _ = file.Close() }, nil
}

func replaceChainFileOSV1(temporary, destination string) error {
	return os.Rename(temporary, destination)
}

func openChainFileOSV1(filename string) (*os.File, error) { return os.Open(filename) }
