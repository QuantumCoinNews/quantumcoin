package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrStorageLocked = errors.New("persistence storage is locked")

// FileLock owns an OS lock; its persistent sidecar is a rendezvous inode.
type FileLock struct {
	file   *os.File
	unlock func() error
	once   sync.Once
	err    error
}

func AcquireFileLock(targetPath string, timeout time.Duration) (*FileLock, error) {
	if targetPath == "" {
		return nil, errors.New("lock target path required")
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(targetPath+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		unlock, err := tryStorageLock(f)
		if err == nil {
			return &FileLock{file: f, unlock: unlock}, nil
		}
		if !errors.Is(err, ErrStorageLocked) || timeout <= 0 || !time.Now().Before(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("acquire storage lock: %w", err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}
func (lock *FileLock) Release() error {
	if lock == nil {
		return nil
	}
	lock.once.Do(func() {
		lock.err = lock.unlock()
		if err := lock.file.Close(); lock.err == nil {
			lock.err = err
		}
	})
	return lock.err
}
