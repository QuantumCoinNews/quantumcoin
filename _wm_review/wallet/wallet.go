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

// PubKey -> HASH160 -> Base58Check (version=0x00)
func GetAddressFromPub(pub []byte) string {
	pubKeyHash := HashPubKey(pub)
	versioned := append([]byte{0x00}, pubKeyHash...)
	checksum := utils.Checksum(versioned)
	full := append(versioned, checksum...)
	return string(utils.Base58Encode(full))
}

func (w *Wallet) GetAddress() string { return GetAddressFromPub(w.PublicKey) }

// PubKey HASH160
func HashPubKey(pubKey []byte) []byte { return utils.Hash160(pubKey) }

// Özel anahtarı hex dışa aktar (ham 32 bayt secp256k1 skalar D)
// ImportPrivateKeyHex bu formatı doğrudan kabul eder.
func (w *Wallet) ExportPrivateKeyHex() string {
	d := w.PrivateKey.D.Bytes()
	out := make([]byte, 32)
	copy(out[32-len(d):], d) // left-pad
	return hex.EncodeToString(out)
}

// Adrese göre cüzdan yükle (depoda varsa onu döndür, yoksa yeni üret)
func LoadWallet(address string) *Wallet {
	w := LoadWalletFromFile()
	if addr := w.GetAddress(); addr == address {
		return w
	}
	return NewWallet()
}

// Yardımcılar
func GetPubKeyHash(pubKey []byte) []byte { return HashPubKey(pubKey) }
func HashAndEncode(pubKey []byte) string { return GetAddressFromPub(pubKey) }

// 65B uncompressed public key (0x04||X||Y)
func (w *Wallet) UncompressedPub() []byte { return w.PublicKey }
