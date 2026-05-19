package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"log"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// KeyPair structı, özel ve açık anahtarları içerir.
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

// NewKeyPair: Yeni ECDSA anahtar çifti üretir.
// QuantumCoin wallet standardı: secp256k1.
// PublicKey formatı: uncompressed 65 byte, 0x04 || X(32) || Y(32).
func NewKeyPair() *KeyPair {
	priv, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		log.Panicf("Anahtar çifti oluşturulamadı: %v", err)
	}

	pub := make([]byte, 65)
	pub[0] = 0x04

	xBytes := priv.PublicKey.X.Bytes()
	yBytes := priv.PublicKey.Y.Bytes()

	copy(pub[1+32-len(xBytes):33], xBytes)
	copy(pub[33+32-len(yBytes):65], yBytes)

	return &KeyPair{
		PrivateKey: priv,
		PublicKey:  pub,
	}
}
