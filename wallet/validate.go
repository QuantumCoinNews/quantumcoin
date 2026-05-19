package wallet

import (
	"bytes"
	"crypto/sha256"

	"quantumcoin/utils"
)

// ValidateAddress: Adres QuantumCoin Base58Check formatında mı?
// Beklenen format:
// [version:1 byte][pubKeyHash:20 byte][checksum:4 byte] = 25 byte
func ValidateAddress(address string) bool {
	decoded, err := utils.Base58Decode([]byte(address))
	if err != nil {
		return false
	}

	if len(decoded) != 25 {
		return false
	}

	version := decoded[0]
	if version != 0x00 {
		return false
	}

	payload := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]
	expectedChecksum := calculateChecksum(payload)

	return bytes.Equal(checksum, expectedChecksum)
}

// Base58 adres için checksum hesaplama.
func calculateChecksum(payload []byte) []byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return second[:4]
}
