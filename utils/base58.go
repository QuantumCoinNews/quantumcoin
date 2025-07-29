package utils

import (
	"bytes"
	"errors"
	"math/big"
)

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// Base58Encode encodes a byte slice to Base58 format
func Base58Encode(input []byte) []byte {
	var result []byte
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	mod := new(big.Int)

	for x.Cmp(big.NewInt(0)) > 0 {
		x.DivMod(x, base, mod)
		result = append(result, b58Alphabet[mod.Int64()])
	}

	// Add '1' for each leading 0 byte
	for _, b := range input {
		if b == 0x00 {
			result = append(result, b58Alphabet[0])
		} else {
			break
		}
	}

	reverse(result)
	return result
}

// Base58Decode decodes a Base58-encoded byte slice
func Base58Decode(input []byte) ([]byte, error) {
	result := big.NewInt(0)
	base := big.NewInt(58)

	for _, b := range input {
		index := bytes.IndexByte(b58Alphabet, b)
		if index == -1 {
			return nil, errors.New("invalid Base58 character")
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(index)))
	}

	decoded := result.Bytes()

	// Add leading zero padding
	nPad := 0
	for _, b := range input {
		if b == b58Alphabet[0] {
			nPad++
		} else {
			break
		}
	}

	return append(bytes.Repeat([]byte{0x00}, nPad), decoded...), nil
}

// reverse reverses a byte slice in-place
func reverse(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}
