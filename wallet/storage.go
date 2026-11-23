package wallet

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrWalletNotFound     = errors.New("wallet not found; explicitly restore recovery words or an encrypted backup, or create a new empty wallet")
	ErrWalletStoreCorrupt = errors.New("wallet storage is invalid; original file preserved; restore from recovery words or an encrypted backup")
	ErrWalletStoreBusy    = errors.New("wallet storage is in use; retry after the other wallet operation completes")
	ErrWalletWouldReplace = errors.New("a different wallet already exists; use a separate wallet profile; existing keys were preserved")
)

const maxWalletStoreBytes = 16 << 20

func readWalletBytes(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxWalletStoreBytes {
		return nil, ErrWalletStoreCorrupt
	}
	f, err := openWalletRead(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxWalletStoreBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || len(raw) > maxWalletStoreBytes {
		return nil, ErrWalletStoreCorrupt
	}
	return raw, nil
}

// Reject duplicate names before decoding so no hidden wallet entry can be
// silently discarded by JSON's usual last-value-wins behaviour.
func decodeWalletJSON(raw []byte, target any) error {
	var shape map[string]json.RawMessage
	if json.Unmarshal(raw, &shape) != nil || shape == nil {
		return ErrWalletStoreCorrupt
	}
	var required string
	allowed := map[string]bool{}
	switch target.(type) {
	case *diskStore:
		required = "wallets"
		allowed["wallets"] = true
		allowed["default"] = true
	case *DefaultWallet:
		required = "address"
		allowed["address"] = true
		allowed["privateKeyHex"] = true
	default:
		return ErrWalletStoreCorrupt
	}
	if _, ok := shape[required]; !ok {
		return ErrWalletStoreCorrupt
	}
	for key := range shape {
		if !allowed[key] {
			return ErrWalletStoreCorrupt
		}
		if key != "wallets" && (len(shape[key]) == 0 || shape[key][0] != '"') {
			return ErrWalletStoreCorrupt
		}
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	var value func(int) error
	value = func(depth int) error {
		if depth > 32 {
			return ErrWalletStoreCorrupt
		}
		token, err := d.Token()
		if err != nil {
			return ErrWalletStoreCorrupt
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				name, err := d.Token()
				if err != nil {
					return ErrWalletStoreCorrupt
				}
				key, ok := name.(string)
				if !ok || seen[key] {
					return ErrWalletStoreCorrupt
				}
				seen[key] = true
				if err = value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err = value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return ErrWalletStoreCorrupt
		}
		end, err := d.Token()
		if err != nil || (delimiter == '{' && end != json.Delim('}')) || (delimiter == '[' && end != json.Delim(']')) {
			return ErrWalletStoreCorrupt
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrWalletStoreCorrupt
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return ErrWalletStoreCorrupt
	}
	return nil
}

// Writers retain this OS lock for the whole read/validate/modify/commit cycle.
// Atomic replacement lets readers observe a complete old or new document.
func lockWalletStore(path string) (func(), error) {
	if path == "" {
		return nil, errors.New("wallet storage path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	return lockWalletFile(path + ".lock")
}

type walletTemporaryFile interface {
	io.Writer
	Sync() error
	Close() error
	Name() string
}
type walletFileOps struct {
	create  func(string) (walletTemporaryFile, error)
	replace func(string, string) error
}

func commitWalletFile(path string, raw []byte, ops walletFileOps) error {
	if len(raw) == 0 || len(raw) > maxWalletStoreBytes {
		return ErrWalletStoreCorrupt
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return ErrWalletStoreCorrupt
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := ops.create(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	n, err := f.Write(raw)
	if err == nil && n != len(raw) {
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
	return ops.replace(f.Name(), path)
}

func writeWalletFileAtomic(path string, raw []byte) error {
	return commitWalletFile(path, raw, walletFileOps{
		create:  func(dir string) (walletTemporaryFile, error) { return os.CreateTemp(dir, ".qc-wallet-store-*") },
		replace: replaceWalletFile,
	})
}
