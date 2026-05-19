package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"log"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// GenerateKeyPair: Yeni anahtar çifti üretir.
// QuantumCoin wallet standardı: secp256k1.
func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privKey, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		log.Panicf("Anahtar çifti oluşturulamadı: %v", err)
	}
	return privKey, &privKey.PublicKey
}

// SignData: Veriyi SHA-256 ile hashleyip imzalar.
func SignData(data []byte, priv *ecdsa.PrivateKey) (*big.Int, *big.Int, error) {
	if priv == nil {
		return nil, nil, errors.New("private key is nil")
	}

	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	return r, s, err
}

// VerifySignature: Verinin imzası doğru mu?
func VerifySignature(pub *ecdsa.PublicKey, data []byte, r, s *big.Int) bool {
	if pub == nil || r == nil || s == nil {
		return false
	}

	hash := sha256.Sum256(data)
	return ecdsa.Verify(pub, hash[:], r, s)
}
