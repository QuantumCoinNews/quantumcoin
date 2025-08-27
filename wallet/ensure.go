package wallet

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultWallet: APPDATA altında JSON olarak saklanan minimal kayıt.
// Not: Burada adres yeterli; istersen PrivateKeyHex'i de tutuyoruz (lokal geliştirme içindir).
type DefaultWallet struct {
	PrivateKeyHex string `json:"privateKeyHex,omitempty"`
	Address       string `json:"address"`
}

// EnsureDefaultWallet, %APPDATA%\QuantumCoin\wallet.json dosyasını garanti eder.
// - Dosya geçerliyse okur ve döndürür.
// - Yoksa yeni bir Wallet üretir, dosyaya yazar ve döndürür.
func EnsureDefaultWallet() (*DefaultWallet, error) {
	dir, err := appDataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "wallet.json")

	// Varsa ve geçerliyse kullan
	if b, err := os.ReadFile(path); err == nil {
		var w DefaultWallet
		if json.Unmarshal(b, &w) == nil && w.Address != "" {
			return &w, nil
		}
		// bozuksa aşağıda yeniden yazacağız
	}

	// Yeni üret
	wal := NewWallet()
	w := &DefaultWallet{
		PrivateKeyHex: safeExportPrivHex(wal),
		Address:       wal.GetAddress(),
	}

	data, _ := json.MarshalIndent(w, "", "  ")
	if err := os.WriteFile(path, data, fs.FileMode(0o600)); err != nil {
		return nil, err
	}
	return w, nil
}

// LoadDefaultWallet mevcut dosyayı okumak için yardımcı (opsiyonel).
func LoadDefaultWallet() (*DefaultWallet, error) {
	dir, err := appDataDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "wallet.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var w DefaultWallet
	if err := json.Unmarshal(b, &w); err != nil {
		return nil, err
	}
	if w.Address == "" {
		return nil, errors.New("wallet.json: address missing")
	}
	return &w, nil
}

// Wallet kaydını güncellemek için (ör. adres değişirse) opsiyonel yardımcı.
func SaveDefaultWallet(w *DefaultWallet) error {
	if w == nil || w.Address == "" {
		return errors.New("invalid wallet")
	}
	dir, err := appDataDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "wallet.json")
	data, _ := json.MarshalIndent(w, "", "  ")
	return os.WriteFile(path, data, fs.FileMode(0o600))
}

// %APPDATA%\QuantumCoin yolunu verir.
func appDataDir() (string, error) {
	app := os.Getenv("APPDATA")
	if app == "" {
		return "", errors.New("APPDATA not set")
	}
	return filepath.Join(app, "QuantumCoin"), nil
}

// Private key export'u güvenli biçimde çağır; yoksa boş bırak.
// (Projende ExportPrivateKeyHex zaten var; main.go'da kullanılıyor.)
func safeExportPrivHex(w *Wallet) string {
	if w == nil {
		return ""
	}
	// ExportPrivateKeyHex mevcut; saklamak istemezsen "" döndürebilirsin.
	return w.ExportPrivateKeyHex()
}
