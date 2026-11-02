// Package utils contains QuantumCoin project code for this module.
package utils

import (
	"crypto/sha256"
	"log"

	//lint:ignore SA1019 QuantumCoin address compatibility requires RIPEMD-160.
	"golang.org/x/crypto/ripemd160"
)

// HashSHA256 documents the QuantumCoin component used by this module.
// HashSHA256: SHA-256 hash
func HashSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// DoubleSHA256 documents the QuantumCoin component used by this module.
// DoubleSHA256: Double SHA-256 (used for blocks, checksums)
func DoubleSHA256(data []byte) []byte {
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	return second[:]
}

// Hash160 documents the QuantumCoin component used by this module.
// Hash160: SHA256 + RIPEMD160 (address/pubkey hash)
func Hash160(data []byte) []byte {
	shaHash := sha256.Sum256(data)
	ripemd := ripemd160.New()
	_, err := ripemd.Write(shaHash[:])
	if err != nil {
		log.Panicf("RIPEMD160 hashing failed: %v", err)
	}
	return ripemd.Sum(nil)
}

// Checksum documents the QuantumCoin component used by this module.
// Checksum: First 4 bytes of DoubleSHA256
func Checksum(payload []byte) []byte {
	fullHash := DoubleSHA256(payload)
	return fullHash[:4]
}
