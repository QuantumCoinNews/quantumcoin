package wallet

import (
	"crypto/elliptic"
	"crypto/x509"
	"encoding/hex"
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
//	  "wallets": { "<address>": "<priv_hex>", ... },
//	  "default": "<address>"
//	}
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
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Geriye dönük uyumluluk: Eski isim
func SaveWalletToFile(w *Wallet) error { return SaveWallet(w) }

// Yeni: cüzdanı kaydet (adres→privHex)
func SaveWallet(w *Wallet) error {
	if w == nil || w.PrivateKey == nil {
		return errors.New("wallet/save: invalid wallet")
	}
	privBytes, err := x509.MarshalECPrivateKey(w.PrivateKey)
	if err != nil {
		return err
	}
	privHex := hex.EncodeToString(privBytes)
	addr := w.GetAddress()

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
		nw := NewWallet()
		_ = SaveWallet(nw)
		return nw
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
	_ = SaveWallet(nw)
	return nw
}

// Belirli adresteki cüzdanı yükle (varsa)
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
	privHex, ok := st.Wallets[address]
	if !ok || privHex == "" {
		return nil, false
	}
	privBytes, err := hex.DecodeString(privHex)
	if err != nil {
		return nil, false
	}
	priv, err := x509.ParseECPrivateKey(privBytes)
	if err != nil {
		return nil, false
	}
	// Uncompressed pubkey (0x04||X||Y), P-256
	pub := make([]byte, 0, 1+2*((priv.Curve.Params().BitSize+7)/8))
	pub = append(pub, 0x04)
	byteLen := (elliptic.P256().Params().BitSize + 7) / 8
	x := priv.PublicKey.X.Bytes()
	y := priv.PublicKey.Y.Bytes()
	if lx := len(x); lx < byteLen {
		pad := make([]byte, byteLen-lx)
		x = append(pad, x...)
	}
	if ly := len(y); ly < byteLen {
		pad := make([]byte, byteLen-ly)
		y = append(pad, y...)
	}
	pub = append(pub, x...)
	pub = append(pub, y...)

	return &Wallet{PrivateKey: priv, PublicKey: pub}, true
}

// Varsayılan adresi işaretle (opsiyonel)
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

// QC adresinden pubKeyHash'i çıkar (Base58Check decode)
func Base58DecodeAddress(address string) []byte {
	decoded, err := utils.Base58Decode([]byte(address))
	if err != nil {
		panic(err) // projedeki eski davranışa uygun
	}
	if len(decoded) < 5 {
		panic("invalid address")
	}
	// decoded: [version][pubKeyHash][checksum]
	return decoded[1 : len(decoded)-4]
}
