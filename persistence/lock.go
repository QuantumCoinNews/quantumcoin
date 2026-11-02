package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrStorageLocked = errors.New("persistence storage is locked")

type FileLock struct {
	path string
	file *os.File
}

func AcquireFileLock(
	targetPath string,
	timeout time.Duration,
) (*FileLock, error) {
	if targetPath == "" {
		return nil, errors.New("lock target path required")
	}

	lockPath := targetPath + ".lock"

	parent := filepath.Dir(lockPath)
	if parent != "." && parent != "" {
		if err := os.MkdirAll(parent, 0o700); err != nil {
			return nil, fmt.Errorf("create lock directory: %w", err)
		}
	}

	deadline := time.Now().Add(timeout)

	for {
		file, err := os.OpenFile(
			lockPath,
			os.O_CREATE|os.O_EXCL|os.O_WRONLY,
			0o600,
		)
		if err == nil {
			return &FileLock{
				path: lockPath,
				file: file,
			}, nil
		}

		if timeout <= 0 || time.Now().After(deadline) {
			return nil, fmt.Errorf("%w: %s", ErrStorageLocked, lockPath)
		}

		time.Sleep(25 * time.Millisecond)
	}
}

func (lock *FileLock) Release() error {
	if lock == nil {
		return nil
	}

	var firstErr error

	if lock.file != nil {
		if err := lock.file.Close(); err != nil {
			firstErr = err
		}
		lock.file = nil
	}

	if err := os.Remove(lock.path); err != nil &&
		!os.IsNotExist(err) &&
		firstErr == nil {
		firstErr = err
	}

	return firstErr
}
