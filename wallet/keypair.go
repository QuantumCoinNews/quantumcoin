package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
)

// KeyPair structı, özel ve açık anahtarları içerir
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

// NewKeyPair: Yeni ECDSA anahtar çifti (uncompressed 65 byte public key ile)
func NewKeyPair() *KeyPair {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Panicf("Anahtar çifti oluşturulamadı: %v", err)
	}
	pub := append([]byte{0x04}, append(priv.PublicKey.X.Bytes(), priv.PublicKey.Y.Bytes()...)...)
	return &KeyPair{
		PrivateKey: priv,
		PublicKey:  pub,
	}
}
