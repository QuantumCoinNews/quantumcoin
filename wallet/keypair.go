// Package wallet contains QuantumCoin project code for this module.
package wallet

import (
	"crypto/ecdsa"
	"log"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// KeyPair structı, özel ve açık anahtarları içerir.
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

// NewKeyPair documents the QuantumCoin component used by this module.
// NewKeyPair: Yeni ECDSA anahtar çifti üretir.
// QuantumCoin wallet standardı: secp256k1.
// PublicKey formatı: uncompressed 65 byte, 0x04 || X(32) || Y(32).
func NewKeyPair() *KeyPair {
	private, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		log.Panicf("key generation failed: %v", err)
	}
	defer private.Zero()
	return &KeyPair{PrivateKey: private.ToECDSA(), PublicKey: private.PubKey().SerializeUncompressed()}
}
