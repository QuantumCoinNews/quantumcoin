package utils

import (
	"crypto/sha256"
	"log"

	"golang.org/x/crypto/ripemd160"
)

// HashSHA256: SHA-256 hash
func HashSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// DoubleSHA256: 2 kez SHA-256 hash (genellikle blok ve checksum için)
func DoubleSHA256(data []byte) []byte {
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	return second[:]
}

// Hash160: SHA256 + RIPEMD160 kombinasyonu (adres, pubkey hash için)
func Hash160(data []byte) []byte {
	shaHash := sha256.Sum256(data)
	ripemd := ripemd160.New()
	_, err := ripemd.Write(shaHash[:])
	if err != nil {
		log.Panicf("RIPEMD160 hashing failed: %v", err)
	}
	return ripemd.Sum(nil)
}

// Checksum: DoubleSHA256'in ilk 4 baytı (adres doğrulama vs. için)
func Checksum(payload []byte) []byte {
	fullHash := DoubleSHA256(payload)
	return fullHash[:4]
}
