//go:build windows

package blockchain

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"golang.org/x/sys/windows"
)

func lockChainFileOSV1(_ string, key string) (func(), error) {
	name, err := windows.UTF16PtrFromString(`Global\QuantumCoin.ChainPersistence.` + key)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		if handle != 0 {
			_ = windows.CloseHandle(handle)
		}
		return nil, err
	}
	// Windows mutex ownership is thread-affine, unlike a Go sync.Mutex.
	runtime.LockOSThread()
	result, err := windows.WaitForSingleObject(handle, windows.INFINITE)
	if err != nil || result != windows.WAIT_OBJECT_0 && result != windows.WAIT_ABANDONED {
		runtime.UnlockOSThread()
		_ = windows.CloseHandle(handle)
		return nil, fmt.Errorf("wait for chain mutex: result=%d error=%v", result, err)
	}
	return func() { _ = windows.ReleaseMutex(handle); _ = windows.CloseHandle(handle); runtime.UnlockOSThread() }, nil
}

func replaceChainFileOSV1(temporary, destination string) error {
	from, err := windows.UTF16PtrFromString(temporary)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	// Readers outside our share-delete aware loader (including indexers and
	// scanners) can briefly prevent replacement. Retry only Windows sharing/
	// access failures; never delete the existing chain as a workaround.
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
		if err == nil {
			return nil
		}
		transient := errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_ACCESS_DENIED)
		if !transient || !time.Now().Before(deadline) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func openChainFileOSV1(filename string) (*os.File, error) {
	path, err := windows.UTF16PtrFromString(filename)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(path, windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), filename), nil
}
