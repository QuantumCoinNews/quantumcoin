// Package wallet contains QuantumCoin project code for this module.
package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"log"

	"quantumcoin/utils"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	// 65 byte uncompressed public key: 0x04 || X(32) || Y(32)
	PublicKey []byte
}

// 32-byte left-pad
func pad32(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// NewWallet documents the QuantumCoin component used by this module.
// Yeni anahtar çifti (secp256k1)
func NewWallet() *Wallet {
	curve := secp256k1.S256()
	priv, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panicf("Cüzdan anahtarı oluşturulamadı: %v", err)
	}
	pub := append([]byte{0x04}, pad32(priv.PublicKey.X.Bytes())...)
	pub = append(pub, pad32(priv.PublicKey.Y.Bytes())...)
	return &Wallet{PrivateKey: priv, PublicKey: pub}
}

// GetAddressFromPub documents the QuantumCoin component used by this module.
// PubKey -> HASH160 -> Base58Check (version=0x00)
func GetAddressFromPub(pub []byte) string {
	pubKeyHash := HashPubKey(pub)
	versioned := append([]byte{0x00}, pubKeyHash...)
	checksum := utils.Checksum(versioned)
	full := append(versioned, checksum...)
	return string(utils.Base58Encode(full))
}

func (w *Wallet) GetAddress() string { return GetAddressFromPub(w.PublicKey) }

// HashPubKey documents the QuantumCoin component used by this module.
// PubKey HASH160
func NormalizePubKey(pubKey []byte) []byte {
	if len(pubKey) == 65 && pubKey[0] == 0x04 {
		return pubKey[1:]
	}
	return pubKey
}

func HashPubKey(pubKey []byte) []byte {
	return utils.Hash160(NormalizePubKey(pubKey))
}

// ExportPrivateKeyHex documents the QuantumCoin component used by this module.
// Özel anahtarı hex dışa aktar (ham 32 bayt secp256k1 skalar D)
// ImportPrivateKeyHex bu formatı doğrudan kabul eder.
func (w *Wallet) ExportPrivateKeyHex() string {
	d := w.PrivateKey.D.Bytes()
	out := make([]byte, 32)
	copy(out[32-len(d):], d) // left-pad
	return hex.EncodeToString(out)
}

// LoadWallet adrese göre cüzdan yükler.
// Güvenlik notu:
// - address boşsa varsayılan cüzdanı döndürür.
// - address doluysa sadece depoda varsa döndürür.
// - adres bulunamazsa sessizce yeni cüzdan üretmez; nil döndürür.
func LoadWallet(address string) *Wallet {
	if address == "" {
		return LoadWalletFromFile()
	}

	if w, ok := LoadWalletByAddress(address); ok {
		return w
	}

	return nil
}

// GetPubKeyHash documents the QuantumCoin component used by this module.
// Yardımcılar
func GetPubKeyHash(pubKey []byte) []byte { return HashPubKey(pubKey) }
func HashAndEncode(pubKey []byte) string { return GetAddressFromPub(pubKey) }

// UncompressedPub documents the QuantumCoin component used by this module.
// 65B uncompressed public key (0x04||X||Y)
func (w *Wallet) UncompressedPub() []byte { return w.PublicKey }
