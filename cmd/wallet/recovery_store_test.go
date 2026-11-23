package main

import (
	"bytes"
	"os"
	"path/filepath"
	"quantumcoin/utils"
	qwallet "quantumcoin/wallet"
	"strings"
	"testing"
	"time"
)

func TestLegacyBackupRestoreKeepsSecretsEncryptedAndOriginals(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	node := t.TempDir()
	t.Setenv("QC_NODE_DIR", node)
	defer lockRecoveryWallet()
	old, restored := qwallet.NewWallet(), qwallet.NewWallet()
	path := filepath.Join(node, "wallet_priv.hex")
	original := []byte(old.ExportPrivateKeyHex())
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "legacy.qcbak")
	password := "synthetic-legacy-password-123"
	secret := restored.ExportPrivateKeyHex()
	words := "unverified legacy notes stay encrypted"
	if err := encryptJSONToFile(backup, map[string]interface{}{"wallet_address": restored.GetAddress(), "wallet_priv_hex": secret, "wallet_recovery_words": words}, password); err != nil {
		t.Fatal(err)
	}
	if err := restoreWalletBackup(backup, password); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, original) {
		t.Fatal("legacy restore overwrote existing private key")
	}
	if _, err := os.Stat(filepath.Join(node, "wallet_recovery_words.txt")); !os.IsNotExist(err) {
		t.Fatal("legacy restore published plaintext recovery words")
	}
	selection, err := selectedRecovery()
	if err != nil || selection == nil {
		t.Fatal("legacy encrypted profile was not selected")
	}
	home, _ := recoveryHome()
	blob, err := os.ReadFile(filepath.Join(home, selection.File))
	if err != nil || bytes.Contains(blob, []byte(secret)) || bytes.Contains(blob, []byte(words)) {
		t.Fatal("legacy profile was not encrypted")
	}
	if detectExistingAddress() != restored.GetAddress() || detectPrivHex() != secret {
		t.Fatal("legacy profile cannot sign for restored address")
	}
	lockRecoveryWallet()
	if detectPrivHex() != "" {
		t.Fatal("locked legacy profile fell back to another private key")
	}
	if unlockRecoveryWallet("wrong-password") == nil {
		t.Fatal("wrong legacy password was accepted")
	}
	if err := unlockRecoveryWallet(password); err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(t.TempDir(), "encrypted-copy.qcbak")
	if err := createWalletBackup(copyPath, password); err != nil {
		t.Fatal(err)
	}
	if err := createWalletBackup(copyPath, password); err == nil {
		t.Fatal("existing legacy backup was overwritten")
	}
	copyBytes, _ := os.ReadFile(copyPath)
	if !bytes.Equal(copyBytes, blob) {
		t.Fatal("backup changed encrypted recovery material")
	}
	// Reinstall/device simulation: only encrypted backup + password move to a
	// different user data directory. No original plaintext runtime is needed.
	lockRecoveryWallet()
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("QC_NODE_DIR", t.TempDir())
	if err := restoreWalletBackup(copyPath, password); err != nil {
		t.Fatal(err)
	}
	if detectPrivHex() != secret || detectExistingAddress() != restored.GetAddress() {
		t.Fatal("portable legacy recovery changed wallet")
	}
	recoverySession.Lock()
	key := recoverySession.legacy.PrivateKey
	recoverySession.expires = time.Now().Add(-time.Second)
	recoverySession.Unlock()
	if detectPrivHex() != "" {
		t.Fatal("expired legacy session retained a usable private key")
	}
	raw, keyErr := utils.SecpPrivateBytesV1(key)
	clear(raw)
	if keyErr == nil {
		t.Fatal("expired legacy key remained exportable")
	}
	if err := selectLegacyRecoveryProfile(); err != nil {
		t.Fatal(err)
	}
	if err := restoreWalletBackup(copyPath, password); err != nil {
		t.Fatal("identical encrypted legacy profile could not be reselected")
	}
}

func TestLegacyBackupRejectsMismatchedAddressWithoutWriting(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	node := t.TempDir()
	t.Setenv("QC_NODE_DIR", node)
	w, other := qwallet.NewWallet(), qwallet.NewWallet()
	backup := filepath.Join(t.TempDir(), "mismatch.qcbak")
	password := "synthetic-legacy-password-123"
	if err := encryptJSONToFile(backup, map[string]string{"wallet_address": other.GetAddress(), "wallet_priv_hex": w.ExportPrivateKeyHex()}, password); err != nil {
		t.Fatal(err)
	}
	if err := restoreWalletBackup(backup, password); err == nil {
		t.Fatal("legacy address/key mismatch accepted")
	}
	files, _ := os.ReadDir(node)
	if len(files) != 0 {
		t.Fatal("invalid legacy backup wrote runtime files")
	}
}

func TestLegacyBackupCreationRequiresRecoverableKey(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("QC_NODE_DIR", t.TempDir())
	target := filepath.Join(t.TempDir(), "empty.qcbak")
	if err := createWalletBackup(target, "synthetic-legacy-password-123"); err == nil {
		t.Fatal("empty legacy backup reported success")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("empty legacy backup created output")
	}
}

func TestRecoveryProfilePreservesWalletsAndLocks(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	defer lockRecoveryWallet()
	m, err := qwallet.RestoreRecoveryMaterial(strings.Repeat("abandon ", 11)+"about", "")
	if err != nil {
		t.Fatal(err)
	}
	const password = "test-only-local-password-123"
	if err = selectRecoveryMaterial(m, password); err != nil {
		t.Fatal(err)
	}
	first, _ := selectedRecovery()
	home, _ := recoveryHome()
	path := filepath.Join(home, first.File)
	before, _ := os.ReadFile(path)
	if unlockedRecoveryKey() == "" {
		t.Fatal("selected wallet not unlocked")
	}
	lockRecoveryWallet()
	if unlockedRecoveryKey() != "" {
		t.Fatal("locked wallet exported a key")
	}
	if unlockRecoveryWallet("wrong-password-123") == nil {
		t.Fatal("wrong password accepted")
	}
	if err = unlockRecoveryWallet(password); err != nil {
		t.Fatal(err)
	}
	if detectExistingAddress() != m.Address {
		t.Fatal("selected address drift")
	}
	backup := filepath.Join(t.TempDir(), "wallet.qcvault")
	if err = createWalletBackup(backup, "incorrect-password"); err == nil {
		t.Fatal("backup accepted wrong password")
	}
	if err = createWalletBackup(backup, password); err != nil {
		t.Fatal(err)
	}
	backupBytes, err := os.ReadFile(backup)
	if err != nil || strings.Contains(string(backupBytes), "abandon") {
		t.Fatal("backup not encrypted")
	}
	// Restoring words with a new local password retains the exact QC address.
	restored, _ := qwallet.RestoreRecoveryMaterial(m.Mnemonic, m.Address)
	if err = selectRecoveryMaterial(restored, password+"new"); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatal("old encrypted wallet overwritten")
	}
	if err = selectLegacyRecoveryProfile(); err != nil {
		t.Fatal(err)
	}
	if selected, err := selectedRecovery(); err != nil || selected != nil {
		t.Fatal("legacy profile not selected")
	}
	after, _ = os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatal("switch deleted encrypted recovery")
	}
	if err = restoreWalletBackup(backup, password); err != nil {
		t.Fatal(err)
	}
	if detectExistingAddress() != m.Address {
		t.Fatal("encrypted backup restored another address")
	}
}

func TestCorruptRecoverySelectionFailsClosed(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	home, _ := recoveryHome()
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "active.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := selectedRecovery(); err == nil {
		t.Fatal("corrupt selection accepted")
	}
	if detectExistingAddress() != "" || detectPrivHex() != "" {
		t.Fatal("corrupt selection fell back to another wallet")
	}
	after, _ := os.ReadFile(path)
	if string(after) != "broken" {
		t.Fatal("corrupt file overwritten")
	}
}
