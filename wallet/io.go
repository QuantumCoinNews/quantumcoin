package wallet

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	path := walletFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &diskStore{Wallets: map[string]string{}, Default: ""}, nil
		}
		return nil, err
	}

	var st diskStore
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	if st.Wallets == nil {
		st.Wallets = map[string]string{}
	}
	return &st, nil
}

func writeStore(st *diskStore) error {
	path := walletFilePath()
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}

	// Private key içeren dosya olduğu için 0600 kullanıyoruz.
	return os.WriteFile(path, data, 0o600)
}

// Geriye dönük uyumluluk: Eski isim.
func SaveWalletToFile(w *Wallet) error { return SaveWallet(w) }

// Yeni: cüzdanı kaydet.
// Store formatı: address -> raw 32-byte private key hex.
func SaveWallet(w *Wallet) error {
	if w == nil || w.PrivateKey == nil {
		return errors.New("wallet/save: invalid wallet")
	}

	addr := w.GetAddress()
	privHex := w.ExportPrivateKeyHex()
	if addr == "" || privHex == "" {
		return errors.New("wallet/save: empty address or private key")
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	st, err := readStore()
	if err != nil {
		return err
	}
	if st.Wallets == nil {
		st.Wallets = map[string]string{}
	}

	st.Wallets[addr] = privHex
	if st.Default == "" {
		st.Default = addr
	}

	return writeStore(st)
}

// Depodan cüzdan yükle:
// - "default" varsa onu döndürür
// - yoksa ilk bulduğunu döndürür
// - hiç yoksa yeni üretip kaydeder
func LoadWalletFromFile() *Wallet {
	storeMu.Lock()
	defer storeMu.Unlock()

	st, err := readStore()
	if err != nil {
		st = &diskStore{Wallets: map[string]string{}, Default: ""}
	}

	if st.Default != "" {
		if w, ok := loadWalletByAddress(st, st.Default); ok {
			return w
		}
	}

	for addr := range st.Wallets {
		if w, ok := loadWalletByAddress(st, addr); ok {
			st.Default = addr
			_ = writeStore(st)
			return w
		}
	}

	nw := NewWallet()
	addr := nw.GetAddress()
	privHex := nw.ExportPrivateKeyHex()

	if st.Wallets == nil {
		st.Wallets = map[string]string{}
	}
	st.Wallets[addr] = privHex
	st.Default = addr
	_ = writeStore(st)

	return nw
}

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

	return &Wallet{PrivateKey: priv, PublicKey: pub}, true
}

// Varsayılan adresi işaretle.
func SetDefaultWallet(address string) error {
	storeMu.Lock()
	defer storeMu.Unlock()

	st, err := readStore()
	if err != nil {
		return err
	}
	if _, ok := st.Wallets[address]; !ok {
		return errors.New("wallet not found in store")
	}
	st.Default = address
	return writeStore(st)
}

// QC adresinden pubKeyHash'i çıkar.
// Eski davranışı korumak için hata durumunda panic eder.
func Base58DecodeAddress(address string) []byte {
	pubKeyHash, err := Base58DecodeAddressSafe(address)
	if err != nil {
		panic(err)
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
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}

	out := make([]byte, 65)
	out[0] = 0x04

	xBytes := pub.X.Bytes()
	yBytes := pub.Y.Bytes()

	if len(xBytes) > 32 || len(yBytes) > 32 {
		return nil
	}

	copy(out[1+32-len(xBytes):33], xBytes)
	copy(out[33+32-len(yBytes):65], yBytes)

	return out
}
