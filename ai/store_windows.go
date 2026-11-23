//go:build windows

package ai

import (
	"crypto/sha256"
	"fmt"
	"golang.org/x/sys/windows"
	"path/filepath"
	"runtime"
	"strings"
)

// Named mutexes also serialize read/modify/write history across processes.
func lockRecordDirectory(dir string) (func(), error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(absolute))))
	name, err := windows.UTF16PtrFromString(fmt.Sprintf("Local\\QuantumCoinAI-%x", digest))
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil && err != windows.ERROR_ALREADY_EXISTS {
		if handle != 0 {
			_ = windows.CloseHandle(handle)
		}
		return nil, err
	}
	runtime.LockOSThread()
	result, err := windows.WaitForSingleObject(handle, windows.INFINITE)
	if err != nil || (result != windows.WAIT_OBJECT_0 && result != windows.WAIT_ABANDONED) {
		runtime.UnlockOSThread()
		_ = windows.CloseHandle(handle)
		return nil, fmt.Errorf("AI store mutex: result=%d error=%v", result, err)
	}
	return func() { _ = windows.ReleaseMutex(handle); _ = windows.CloseHandle(handle); runtime.UnlockOSThread() }, nil
}

func replaceRecord(from, to string) error {
	src, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	dst, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(src, dst, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
