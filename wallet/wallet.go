package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"log"

	"quantumcoin/utils"
)

// Wallet bir cüzdanın özel ve açık anahtarlarını tutar
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte // Uncompressed: 0x04 || X || Y
}

// Yeni anahtar çifti üret
func NewWallet() *Wallet {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panicf("Cüzdan anahtarı oluşturulamadı: %v", err)
	}
	// Uncompressed public key: 0x04 || X || Y
	pub := append([]byte{0x04}, append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)...)
	return &Wallet{PrivateKey: privKey, PublicKey: pub}
}

// Base58Check adres üretimi
func (w *Wallet) GetAddress() string {
	pubKeyHash := HashPubKey(w.PublicKey)
	versionedPayload := append([]byte{0x00}, pubKeyHash...) // 0x00 versiyon
	checksum := utils.Checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	return string(utils.Base58Encode(fullPayload))
}

// PubKey -> HASH160
func HashPubKey(pubKey []byte) []byte {
	return utils.Hash160(pubKey)
}

// Özel anahtarı hex dışa aktar (dev/test)
func (w *Wallet) ExportPrivateKeyHex() string {
	privBytes, err := x509.MarshalECPrivateKey(w.PrivateKey)
	if err != nil {
		log.Panicf("Özel anahtar dışa aktarılamadı: %v", err)
	}
	return hex.EncodeToString(privBytes)
}

// Adrese göre cüzdan yükle (depoda varsa onu döndür, yoksa yeni üret)
func LoadWallet(address string) *Wallet {
	w := LoadWalletFromFile() // depodan default/ilk cüzdan
	if addr := w.GetAddress(); addr == address {
		return w
	}
	// eşleşmiyorsa yeni cüzdan (eski davranış)
	return NewWallet()
}

// Yardımcılar
func GetPubKeyHash(pubKey []byte) []byte { return HashPubKey(pubKey) }

func HashAndEncode(pubKey []byte) string {
	pubKeyHash := HashPubKey(pubKey)
	versionedPayload := append([]byte{0x00}, pubKeyHash...)
	checksum := utils.Checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	return string(utils.Base58Encode(fullPayload))
}
