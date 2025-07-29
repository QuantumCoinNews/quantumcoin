package wallet

import (
	"bytes"
	"crypto/sha256"

	"quantumcoin/utils"
)

// ValidateAddress: Adres Base58Check formatında ve checksum doğru mu?
func ValidateAddress(address string) bool {
	decoded, err := utils.Base58Decode([]byte(address))
	if err != nil || len(decoded) < 5 {
		return false
	}
	payload := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]
	expectedChecksum := calculateChecksum(payload)
	return bytes.Equal(checksum, expectedChecksum)
}

// Base58 adres için checksum hesaplama
func calculateChecksum(payload []byte) []byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return second[:4]
}
