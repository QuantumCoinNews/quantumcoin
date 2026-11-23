package blockchain

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestChainReplaceWaitsForTransientWindowsReader(t *testing.T) {
	root := t.TempDir()
	destination, temporary := filepath.Join(root, "chain.dat"), filepath.Join(root, "next.tmp")
	if err := os.WriteFile(destination, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(temporary, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	path, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(path, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	held := true
	defer func() {
		if held {
			_ = windows.CloseHandle(handle)
		}
	}()
	done := make(chan error, 1)
	go func() { done <- replaceChainFileOSV1(temporary, destination) }()
	select {
	case err := <-done:
		t.Fatalf("replacement failed instead of waiting for transient reader: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "old" {
		t.Fatal("old destination changed while reader held")
	}
	if err := windows.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	held = false
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("replacement did not finish after reader release")
	}
	data, err = os.ReadFile(destination)
	if err != nil || string(data) != "new" {
		t.Fatal("new file was not installed")
	}
}
