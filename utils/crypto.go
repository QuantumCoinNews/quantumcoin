package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"math/big"
)

// curve: Elliptic curve selection for signing/verification
func curve() elliptic.Curve {
	return elliptic.P256()
	// For secp256k1: use github.com/btcsuite/btcd/btcec
}

// VerifySig: ECDSA signature verification
func VerifySig(pubKey []byte, hash []byte, rBytes []byte, sBytes []byte) bool {
	if len(pubKey) != 65 || pubKey[0] != 0x04 {
		return false
	}
	x := new(big.Int).SetBytes(pubKey[1:33])
	y := new(big.Int).SetBytes(pubKey[33:65])
	publicKey := ecdsa.PublicKey{Curve: curve(), X: x, Y: y}
	r := new(big.Int).SetBytes(rBytes)
	s := new(big.Int).SetBytes(sBytes)
	return ecdsa.Verify(&publicKey, hash, r, s)
}

// HashData: SHA256 hashing
func HashData(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
