// Package wallet contains QuantumCoin project code for this module.
package wallet

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// DefaultWallet documents the QuantumCoin component used by this module.
// DefaultWallet: APPDATA altında JSON olarak saklanan minimal kayıt.
// Not: Burada adres yeterli; istersen PrivateKeyHex'i de tutuyoruz (lokal geliştirme içindir).
type DefaultWallet struct {
	PrivateKeyHex string `json:"privateKeyHex,omitempty"`
	Address       string `json:"address"`
}

// EnsureDefaultWallet loads an existing valid profile. Despite its legacy
// name, it never generates a replacement for missing, unreadable or corrupt data.
// Creation/restoration is explicit through the QC recovery UI or SaveDefaultWallet.
func EnsureDefaultWallet() (*DefaultWallet, error) { return LoadDefaultWallet() }

func validateDefaultWallet(w *DefaultWallet) error {
	if w == nil || !ValidateAddress(w.Address) {
		return ErrWalletStoreCorrupt
	}
	if w.PrivateKeyHex != "" {
		key, err := ImportPrivateKeyHex(w.PrivateKeyHex)
		if err != nil || key == nil {
			return ErrWalletStoreCorrupt
		}
		if GetAddressFromPub(publicKeyBytes(&key.PublicKey)) != w.Address {
			return ErrWalletStoreCorrupt
		}
	}
	return nil
}

func readDefaultWalletAt(path string) (*DefaultWallet, error) {
	raw, err := readWalletBytes(path)
	if err != nil {
		return nil, err
	}
	defer clear(raw)
	var w DefaultWallet
	if err = decodeWalletJSON(raw, &w); err != nil {
		return nil, err
	}
	if err = validateDefaultWallet(&w); err != nil {
		return nil, err
	}
	return &w, nil
}

func LoadDefaultWallet() (*DefaultWallet, error) {
	dir, err := appDataDir()
	if err != nil {
		return nil, err
	}
	storeMu.Lock()
	defer storeMu.Unlock()
	w, err := readDefaultWalletAt(filepath.Join(dir, "wallet.json"))
	if os.IsNotExist(err) {
		return nil, ErrWalletNotFound
	}
	return w, err
}

// SaveDefaultWallet is explicit, preserves any existing key and refuses a
// different-address replacement. New wallets belong in separate profiles.
func SaveDefaultWallet(w *DefaultWallet) error {
	if err := validateDefaultWallet(w); err != nil {
		return err
	}
	dir, err := appDataDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "wallet.json")
	storeMu.Lock()
	defer storeMu.Unlock()
	unlock, err := lockWalletStore(path)
	if err != nil {
		return err
	}
	defer unlock()
	next := *w
	existing, err := readDefaultWalletAt(path)
	if err == nil {
		if existing.Address != next.Address {
			return ErrWalletWouldReplace
		}
		if existing.PrivateKeyHex != "" {
			next.PrivateKeyHex = existing.PrivateKeyHex
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	defer clear(raw)
	return writeWalletFileAtomic(path, raw)
}

// %APPDATA%\QuantumCoin yolunu verir.
func appDataDir() (string, error) {
	app := os.Getenv("APPDATA")
	if app == "" {
		return "", errors.New("APPDATA not set")
	}
	return filepath.Join(app, "QuantumCoin"), nil
}
