package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"quantumcoin/utils"
	"strings"
	"sync"
	"time"

	qwallet "quantumcoin/wallet"
)

const legacyRecoverySelectionVersion = "QC_ACTIVE_LEGACY_BACKUP_V1"

type recoverySelection struct {
	Version string `json:"version"`
	Address string `json:"address"`
	File    string `json:"file"`
}

var recoverySession struct {
	sync.Mutex
	material *qwallet.RecoveryMaterial
	legacy   *qwallet.Wallet
	expires  time.Time
	timer    *time.Timer
	serial   uint64
}

func recoveryHome() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", errors.New("local wallet directory unavailable")
	}
	return filepath.Join(base, "QuantumCoin", "recovery-wallets"), nil
}

func readRecoveryFile(path string, maximum int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maximum+1))
	if err != nil || int64(len(b)) > maximum {
		return nil, errors.New("wallet file is too large or unreadable")
	}
	return b, nil
}

func selectedRecovery() (*recoverySelection, error) {
	home, err := recoveryHome()
	if err != nil {
		return nil, err
	}
	b, err := readRecoveryFile(filepath.Join(home, "active.json"), 4096)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var selection recoverySelection
	if json.Unmarshal(b, &selection) != nil || !qwallet.ValidateAddress(selection.Address) ||
		selection.File == "" || filepath.Base(selection.File) != selection.File {
		return nil, errors.New("active wallet metadata is invalid; restore words or import an encrypted backup")
	}
	if !((selection.Version == "QC_ACTIVE_RECOVERY_V1" && filepath.Ext(selection.File) == ".qcvault") ||
		(selection.Version == legacyRecoverySelectionVersion && filepath.Ext(selection.File) == ".qcbak")) {
		return nil, errors.New("unsupported encrypted wallet selection")
	}
	return &selection, nil
}

func selectedRecoveryAddress() (string, error) {
	selection, err := selectedRecovery()
	if err != nil || selection == nil {
		return "", err
	}
	return selection.Address, nil
}

func selectRecoveryMaterial(material *qwallet.RecoveryMaterial, password string) error {
	blob, err := qwallet.SealRecoveryMaterial(*material, password)
	if err != nil {
		return err
	}
	// Verify before publishing either the new vault or its public selection.
	if checked, e := qwallet.OpenRecoveryMaterial(blob, password); e != nil || checked.Address != material.Address {
		return errors.New("could not verify saved recovery wallet")
	}
	home, err := recoveryHome()
	if err != nil {
		return err
	}
	hash := sha256.Sum256(blob)
	name := material.Address + "-" + hex.EncodeToString(hash[:8]) + ".qcvault"
	if err = qwallet.SaveRecoveryVaultNew(filepath.Join(home, name), blob); err != nil {
		return err
	}
	selection := recoverySelection{Version: "QC_ACTIVE_RECOVERY_V1", Address: material.Address, File: name}
	if err := persistRecoverySelection(home, selection); err != nil {
		return err
	}
	setRecoverySession(material)
	return nil
}

func persistRecoverySelection(home string, selection recoverySelection) error {
	data, _ := json.Marshal(selection)
	f, err := os.CreateTemp(home, ".active-*")
	if err != nil {
		return errors.New("wallet saved but could not select it")
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errors.New("wallet saved but active selection could not be persisted")
	}
	if err = os.Rename(f.Name(), filepath.Join(home, "active.json")); err != nil {
		return errors.New("wallet saved but active selection could not be replaced")
	}
	return nil
}

func setRecoverySession(material *qwallet.RecoveryMaterial) {
	setRecoverySessionState(material, nil)
}

func clearRecoverySessionLocked() {
	recoverySession.material = nil
	if recoverySession.legacy != nil {
		utils.ZeroSecpPrivateV1(recoverySession.legacy.PrivateKey)
		recoverySession.legacy = nil
	}
}

func setRecoverySessionState(material *qwallet.RecoveryMaterial, legacy *qwallet.Wallet) {
	recoverySession.Lock()
	defer recoverySession.Unlock()
	clearRecoverySessionLocked()
	if recoverySession.timer != nil {
		recoverySession.timer.Stop()
	}
	recoverySession.serial++
	serial := recoverySession.serial
	recoverySession.material = material
	recoverySession.legacy = legacy
	recoverySession.expires = time.Now().Add(5 * time.Minute)
	recoverySession.timer = time.AfterFunc(5*time.Minute, func() {
		recoverySession.Lock()
		defer recoverySession.Unlock()
		if recoverySession.serial == serial {
			clearRecoverySessionLocked()
		}
	})
}

func lockRecoveryWallet() {
	recoverySession.Lock()
	defer recoverySession.Unlock()
	recoverySession.serial++
	if recoverySession.timer != nil {
		recoverySession.timer.Stop()
	}
	clearRecoverySessionLocked()
}

// Only the public selection is archived. All encrypted vaults and legacy files
// remain available; the user can explicitly switch to the previous profile.
func selectLegacyRecoveryProfile() error {
	home, err := recoveryHome()
	if err != nil {
		return err
	}
	path := filepath.Join(home, "active.json")
	if _, err = os.Stat(path); os.IsNotExist(err) {
		lockRecoveryWallet()
		return nil
	} else if err != nil {
		return err
	}
	archive := filepath.Join(home, fmt.Sprintf("selection-%d.json", time.Now().UnixNano()))
	if err = os.Link(path, archive); err != nil {
		return err
	}
	if err = os.Remove(path); err != nil {
		return err
	}
	lockRecoveryWallet()
	return nil
}

func unlockedRecoveryKey() string {
	recoverySession.Lock()
	defer recoverySession.Unlock()
	if !time.Now().Before(recoverySession.expires) {
		clearRecoverySessionLocked()
		return ""
	}
	address, err := selectedRecoveryAddress()
	if err != nil {
		return ""
	}
	if recoverySession.legacy != nil {
		if address != recoverySession.legacy.GetAddress() {
			return ""
		}
		return recoverySession.legacy.ExportPrivateKeyHex()
	}
	if recoverySession.material == nil || address != recoverySession.material.Address {
		return ""
	}
	w, err := recoverySession.material.Wallet()
	if err != nil {
		return ""
	}
	defer utils.ZeroSecpPrivateV1(w.PrivateKey)
	return w.ExportPrivateKeyHex()
}

func unlockRecoveryWallet(password string) error {
	selection, err := selectedRecovery()
	if err != nil {
		return err
	}
	if selection == nil {
		return errors.New("no encrypted recovery wallet selected")
	}
	home, err := recoveryHome()
	if err != nil {
		return err
	}
	limit := int64(16384)
	if selection.Version == legacyRecoverySelectionVersion {
		limit = 1024 * 1024
	}
	blob, err := readRecoveryFile(filepath.Join(home, selection.File), limit)
	if err != nil {
		return errors.New("encrypted wallet unavailable; restore words or backup")
	}
	if selection.Version == legacyRecoverySelectionVersion {
		legacy, err := openLegacyRecoveryWallet(blob, password)
		if err != nil {
			return err
		}
		if legacy.GetAddress() != selection.Address {
			utils.ZeroSecpPrivateV1(legacy.PrivateKey)
			return errors.New("active wallet address does not match its vault")
		}
		setRecoverySessionState(nil, legacy)
		return nil
	}
	material, err := qwallet.OpenRecoveryMaterial(blob, password)
	if err != nil {
		return err
	}
	if material.Address != selection.Address {
		return errors.New("active wallet address does not match its vault")
	}
	setRecoverySession(material)
	return nil
}

// Legacy backup words may be unrelated to the random key. The validated key
// is the signing authority; unverified notes never become recovery words.
func legacyRecoveryWallet(data map[string]interface{}) (*qwallet.Wallet, error) {
	allowed := map[string]bool{"wallet_address": true, "miner_address": true, "wallet_priv_hex": true,
		"wallet_recovery_words": true, "wallet_recovery_manifest": true, "balance_cache": true}
	for name, value := range data {
		if !allowed[name] {
			return nil, errors.New("unknown legacy wallet backup field")
		}
		if name != "wallet_recovery_manifest" && name != "balance_cache" {
			if _, ok := value.(string); !ok {
				return nil, errors.New("invalid legacy wallet backup field")
			}
		}
	}
	hexKey, _ := data["wallet_priv_hex"].(string)
	hexKey = strings.TrimSpace(hexKey)
	if len(hexKey) == 0 || len(hexKey) > 4096 {
		return nil, errors.New("legacy backup contains no recoverable QC key")
	}
	key, err := qwallet.ImportPrivateKeyHex(hexKey)
	if err != nil {
		return nil, errors.New("invalid legacy QC key")
	}
	pub, err := utils.SecpPublicBytesV1(&key.PublicKey)
	if err != nil {
		utils.ZeroSecpPrivateV1(key)
		return nil, errors.New("invalid legacy QC key")
	}
	w := &qwallet.Wallet{PrivateKey: key, PublicKey: pub}
	address, _ := data["wallet_address"].(string)
	if address != "" && strings.TrimSpace(address) != w.GetAddress() {
		utils.ZeroSecpPrivateV1(key)
		return nil, errors.New("legacy backup address/key mismatch")
	}
	return w, nil
}

func openLegacyRecoveryWallet(blob []byte, password string) (*qwallet.Wallet, error) {
	var data map[string]interface{}
	if err := decryptLegacyRecoveryBlob(blob, password, &data); err != nil {
		return nil, err
	}
	return legacyRecoveryWallet(data)
}

// Publish ciphertext only, without replacing an existing destination.
func saveEncryptedRecoveryBlobNew(path string, blob []byte) error {
	if len(blob) == 0 || len(blob) > 1024*1024 {
		return errors.New("invalid encrypted wallet size")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".qc-encrypted-backup-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	n, err := f.Write(blob)
	if err == nil && n != len(blob) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Link(f.Name(), path); err != nil {
		return errors.New("backup destination exists or cannot be safely created")
	}
	return nil
}

func selectLegacyEncryptedBackup(blob []byte, password string) error {
	legacy, err := openLegacyRecoveryWallet(blob, password)
	if err != nil {
		return err
	}
	selected := false
	defer func() {
		if !selected {
			utils.ZeroSecpPrivateV1(legacy.PrivateKey)
		}
	}()
	home, err := recoveryHome()
	if err != nil {
		return err
	}
	hash := sha256.Sum256(blob)
	name := legacy.GetAddress() + "-" + hex.EncodeToString(hash[:8]) + ".qcbak"
	path := filepath.Join(home, name)
	existing, readErr := readRecoveryFile(path, 1024*1024)
	if readErr == nil {
		if !bytes.Equal(existing, blob) {
			return errors.New("existing encrypted legacy wallet differs")
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	} else if err = saveEncryptedRecoveryBlobNew(path, blob); err != nil {
		return err
	}
	if err = persistRecoverySelection(home, recoverySelection{Version: legacyRecoverySelectionVersion, Address: legacy.GetAddress(), File: name}); err != nil {
		return err
	}
	setRecoverySessionState(nil, legacy)
	selected = true
	return nil
}
