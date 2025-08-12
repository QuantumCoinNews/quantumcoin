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

// Wallet, bir cüzdanın özel ve açık anahtarlarını tutar
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte // Uncompressed: 0x04 || X || Y
}

// NewWallet yeni bir cüzdan (anahtar çifti) oluşturur
func NewWallet() *Wallet {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panicf("Cüzdan anahtarı oluşturulamadı: %v", err)
	}
	// Uncompressed public key: 0x04 prefix + X + Y
	pub := append([]byte{0x04}, append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)...)
	return &Wallet{PrivateKey: privKey, PublicKey: pub}
}

// GetAddress cüzdanın Base58Check formatında adresini üretir
func (w *Wallet) GetAddress() string {
	pubKeyHash := HashPubKey(w.PublicKey)
	versionedPayload := append([]byte{0x00}, pubKeyHash...) // 0x00 versiyon byte (Bitcoin uyumlu)
	checksum := utils.Checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	return string(utils.Base58Encode(fullPayload))
}

// HashPubKey public key'den public key hash üretir (SHA256 + RIPEMD160)
func HashPubKey(pubKey []byte) []byte {
	return utils.Hash160(pubKey)
}

// ExportPrivateKeyHex özel anahtarı hex formatında dışa aktarır (opsiyonel)
func (w *Wallet) ExportPrivateKeyHex() string {
	privBytes, err := x509.MarshalECPrivateKey(w.PrivateKey)
	if err != nil {
		log.Panicf("Özel anahtar dışa aktarılamadı: %v", err)
	}
	return hex.EncodeToString(privBytes)
}

// LoadWallet, adres bazlı cüzdan yükleme (önce dosyadakini dener, eşleşirse onu döner)
func LoadWallet(address string) *Wallet {
	// 1) Dosyadan tek cüzdanı yükle (mevcut mimari tek wallet tutuyor)
	w := LoadWalletFromFile()
	// 2) Eğer adres eşleşiyorsa bu cüzdanı döndür
	if addr := w.GetAddress(); addr == address {
		return w
	}
	// 3) Eşleşmiyorsa (ya da dosya yoksa) yeni cüzdan oluştur (eski davranış korunur)
	return NewWallet()
}

// GetPubKeyHash, public key'den hash üretir (utils.Hash160 çağrısı)
func GetPubKeyHash(pubKey []byte) []byte {
	return HashPubKey(pubKey)
}

// HashAndEncode: public key'i hash'leyip Base58 encode eder (Explorer için)
func HashAndEncode(pubKey []byte) string {
	pubKeyHash := HashPubKey(pubKey)
	versionedPayload := append([]byte{0x00}, pubKeyHash...)
	checksum := utils.Checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	return string(utils.Base58Encode(fullPayload))
}
