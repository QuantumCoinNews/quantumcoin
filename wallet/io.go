// Package wallet contains QuantumCoin project code for this module.
package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"

	"quantumcoin/config"
	"quantumcoin/utils"
)

var storeMu sync.Mutex

// Disk formatı:
//
//	{
//	 "wallets": { "<address>": "<priv_hex>", ... },
//	 "default": "<address>"
//	}
//
// QuantumCoin wallet standardı:
// - Curve: secp256k1
// - Private key storage: raw 32-byte scalar hex
// - Public key format: uncompressed 65 byte, 0x04 || X(32) || Y(32)
type diskStore struct {
	Wallets map[string]string `json:"wallets"`
	Default string            `json:"default"`
}

func walletFilePath() string {
	cfg := config.Current()
	path := cfg.WalletFile
	if path == "" {
		path = "wallet_data.json"
	}
	return path
}

func readStore() (*diskStore, error) {
	return readStoreAt(walletFilePath())
}

func readStoreAt(path string) (*diskStore, error) {
	data, err := readWalletBytes(path)
	if err != nil {
		return nil, err
	}
	defer clear(data)
	var st diskStore
	if err = decodeWalletJSON(data, &st); err != nil {
		return nil, err
	}
	if err = validateDiskStore(&st); err != nil {
		return nil, err
	}
	return &st, nil
}

func validateDiskStore(st *diskStore) error {
	if st == nil || st.Wallets == nil {
		return ErrWalletStoreCorrupt
	}
	if st.Default != "" {
		if _, ok := st.Wallets[st.Default]; !ok {
			return ErrWalletStoreCorrupt
		}
	}
	for address := range st.Wallets {
		if _, ok := loadWalletByAddress(st, address); !ok {
			return ErrWalletStoreCorrupt
		}
	}
	return nil
}

func writeStoreAt(path string, st *diskStore) error {
	if err := validateDiskStore(st); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	defer clear(raw)
	return writeWalletFileAtomic(path, raw)
}

// SaveWalletToFile documents the QuantumCoin component used by this module.
// Geriye dönük uyumluluk: Eski isim.
func SaveWalletToFile(w *Wallet) error { return SaveWallet(w) }

// SaveWallet documents the QuantumCoin component used by this module.
// Yeni: cüzdanı kaydet.
// Store formatı: address -> raw 32-byte private key hex.
func SaveWallet(w *Wallet) error {
	return SaveWalletAt(walletFilePath(), w)
}

// SaveWalletAt supports local wallet commands without loading node config.
func SaveWalletAt(path string, w *Wallet) error {
	if path == "" {
		return errors.New("wallet/save: path required")
	}
	if w == nil || w.PrivateKey == nil {
		return errors.New("wallet/save: invalid wallet")
	}
	addr, privHex := w.GetAddress(), w.ExportPrivateKeyHex()
	imported, err := ImportPrivateKeyHex(privHex)
	if err != nil || imported == nil || !bytes.Equal(publicKeyBytes(&imported.PublicKey), w.PublicKey) {
		return errors.New("wallet/save: key and address do not match")
	}
	storeMu.Lock()
	defer storeMu.Unlock()

	unlock, err := lockWalletStore(path)
	if err != nil {
		return err
	}
	defer unlock()
	st, err := readStoreAt(path)
	if os.IsNotExist(err) {
		st = &diskStore{Wallets: map[string]string{}}
	} else if err != nil {
		return err
	}
	st.Wallets[addr] = privHex
	if st.Default == "" {
		st.Default = addr
	}

	return writeStoreAt(path, st)
}

// LoadWalletFromFile is the compatibility wrapper. It never creates a wallet
// or modifies a file while loading; nil indicates a missing or invalid store.
func LoadWalletFromFile() *Wallet {
	w, _ := LoadWalletFromFileWithError()
	return w
}

func LoadWalletFromFileWithError() (*Wallet, error) { return LoadStoredWalletAt(walletFilePath()) }

// LoadStoredWalletAt preserves read/permission/format errors for callers that
// need to present recovery options. Loading never writes a new default either.
func LoadStoredWalletAt(path string) (*Wallet, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	st, err := readStoreAt(path)
	if os.IsNotExist(err) {
		return nil, ErrWalletNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(st.Wallets) == 0 {
		return nil, ErrWalletNotFound
	}
	address := st.Default
	if address == "" {
		addresses := make([]string, 0, len(st.Wallets))
		for a := range st.Wallets {
			addresses = append(addresses, a)
		}
		sort.Strings(addresses)
		address = addresses[0]
	}
	w, ok := loadWalletByAddress(st, address)
	if !ok {
		return nil, ErrWalletStoreCorrupt
	}
	return w, nil
}

// LoadWalletByAddress documents the QuantumCoin component used by this module.
// Belirli adresteki cüzdanı yükle.
func LoadWalletByAddress(address string) (*Wallet, bool) {
	storeMu.Lock()
	defer storeMu.Unlock()

	st, err := readStore()
	if err != nil {
		return nil, false
	}
	return loadWalletByAddress(st, address)
}

func loadWalletByAddress(st *diskStore, address string) (*Wallet, bool) {
	if st == nil || st.Wallets == nil {
		return nil, false
	}

	privHex, ok := st.Wallets[address]
	if !ok || privHex == "" {
		return nil, false
	}

	// ImportPrivateKeyHex geriye dönük uyumluluk için eski DER/x509 formatını da,
	// yeni standart olan raw 32-byte secp256k1 hex formatını da kabul eder.
	priv, err := ImportPrivateKeyHex(privHex)
	if err != nil || priv == nil {
		return nil, false
	}

	pub := publicKeyBytes(&priv.PublicKey)
	if len(pub) != 65 {
		return nil, false
	}

	w := &Wallet{PrivateKey: priv, PublicKey: pub}
	if !ValidateAddress(address) || w.GetAddress() != address {
		return nil, false
	}
	return w, true
}

// SetDefaultWallet documents the QuantumCoin component used by this module.
// Varsayılan adresi işaretle.
func SetDefaultWallet(address string) error {
	path := walletFilePath()
	storeMu.Lock()
	defer storeMu.Unlock()

	unlock, err := lockWalletStore(path)
	if err != nil {
		return err
	}
	defer unlock()
	st, err := readStoreAt(path)
	if err != nil {
		return err
	}
	if _, ok := st.Wallets[address]; !ok {
		return errors.New("wallet not found in store")
	}
	st.Default = address
	return writeStoreAt(path, st)
}

// Base58DecodeAddress QC adresinden pubKeyHash'i çıkarır.
//
// INVALID_ADDRESS_NO_PANIC_V1:
// Eski wrapper artık hatalı adreslerde panic etmez. Dışarıdan gelen bozuk
// adresler node'u düşürmemeli; güvenli doğrulama isteyen kod
// Base58DecodeAddressSafe ile error almalıdır.
func Base58DecodeAddress(address string) []byte {
	pubKeyHash, err := Base58DecodeAddressSafe(address)
	if err != nil {
		return nil
	}
	return pubKeyHash
}

// Base58DecodeAddressSafe QC adresinden pubKeyHash'i güvenli şekilde çıkarır.
func Base58DecodeAddressSafe(address string) ([]byte, error) {
	decoded, err := utils.Base58Decode([]byte(address))
	if err != nil {
		return nil, err
	}
	if len(decoded) != 25 {
		return nil, errors.New("invalid address length")
	}

	version := decoded[0]
	if version != 0x00 {
		return nil, errors.New("invalid address version")
	}

	// decoded: [version][pubKeyHash][checksum]
	return decoded[1 : len(decoded)-4], nil
}

func publicKeyBytes(pub *ecdsa.PublicKey) []byte {
	raw, err := utils.SecpPublicBytesV1(pub)
	if err != nil {
		return nil
	}
	return raw
}
