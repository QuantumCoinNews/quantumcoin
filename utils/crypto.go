package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"math/big"
)

// curve: Hangi eğriyle imza/verify yapılacak?
func curve() elliptic.Curve {
	return elliptic.P256()
	// secp256k1 için: github.com/btcsuite/btcd/btcec kullanılır
}

// VerifySig: ECDSA signature doğrulama
// pubKey: 65-byte uncompressed (0x04 || X(32) || Y(32))
// hash: sha256 hash (32 byte)
// rBytes, sBytes: İmza bileşenleri
func VerifySig(pubKey []byte, hash []byte, rBytes []byte, sBytes []byte) bool {
	if len(pubKey) != 65 || pubKey[0] != 0x04 {
		// Sadece uncompressed public key kabul edilir
		return false
	}
	x := new(big.Int).SetBytes(pubKey[1:33])
	y := new(big.Int).SetBytes(pubKey[33:65])
	publicKey := ecdsa.PublicKey{Curve: curve(), X: x, Y: y}
	r := new(big.Int).SetBytes(rBytes)
	s := new(big.Int).SetBytes(sBytes)
	return ecdsa.Verify(&publicKey, hash, r, s)
}

// HashData: SHA256 ile hash al
func HashData(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
