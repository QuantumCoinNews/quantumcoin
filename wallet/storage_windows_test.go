//go:build windows

package wallet

import (
	"bytes"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
)

func TestWalletReadDeniedNeverCreatesReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallet.json")
	if err := SaveWalletAt(path, NewWallet()); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	name, _ := windows.UTF16PtrFromString(path)
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer windows.CloseHandle(handle)
		if w, err := LoadStoredWalletAt(path); w != nil || err == nil {
			t.Fatal("unreadable wallet accepted")
		}
		if err := SaveWalletAt(path, NewWallet()); err == nil {
			t.Fatal("unreadable wallet overwritten")
		}
	}()
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("read failure changed original wallet")
	}
}

func TestWalletReadOnlyDestinationSurvivesFailedReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallet.json")
	if err := SaveWalletAt(path, NewWallet()); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	name, _ := windows.UTF16PtrFromString(path)
	if err := windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_READONLY); err != nil {
		t.Fatal(err)
	}
	defer windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_NORMAL)
	if err := SaveWalletAt(path, NewWallet()); err == nil {
		t.Fatal("read-only save reported success")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed replacement changed original keys")
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".qc-wallet-store-*"))
	if len(leftovers) != 0 {
		t.Fatal("failed replacement left a secret temp file")
	}
}

func TestWalletBusyWriterFailsWithoutChangingExistingKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallet.json")
	if err := SaveWalletAt(path, NewWallet()); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	unlock, err := lockWalletStore(path)
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer unlock()
		if err := SaveWalletAt(path, NewWallet()); !errors.Is(err, ErrWalletStoreBusy) {
			t.Fatal("busy storage was not reported")
		}
	}()
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("busy writer changed existing keys")
	}
	if err := SaveWalletAt(path, NewWallet()); err != nil {
		t.Fatal("released lock still blocks wallet")
	}
}
