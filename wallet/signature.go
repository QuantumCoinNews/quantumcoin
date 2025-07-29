package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
)

// GenerateKeyPair: Yeni anahtar çifti (private & public)
func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return privKey, &privKey.PublicKey
}

// SignData: Veriyi imzala (r, s, error döndürür)
func SignData(data []byte, priv *ecdsa.PrivateKey) (*big.Int, *big.Int, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	return r, s, err
}

// VerifySignature: Verinin imzası doğru mu?
func VerifySignature(pub *ecdsa.PublicKey, data []byte, r, s *big.Int) bool {
	hash := sha256.Sum256(data)
	return ecdsa.Verify(pub, hash[:], r, s)
}
