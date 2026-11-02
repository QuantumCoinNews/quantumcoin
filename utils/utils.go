// Package utils contains QuantumCoin project code for this module.
package utils

import (
	"bytes"
	"encoding/binary"
	"math/big"
)

// Uint64ToBytes documents the QuantumCoin component used by this module.
// Uint64ToBytes: uint64 → []byte (big-endian)
func Uint64ToBytes(num uint64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		panic(err)
	}
	return buff.Bytes()
}

// BytesToUint64 documents the QuantumCoin component used by this module.
// BytesToUint64: []byte → uint64 (big-endian)
func BytesToUint64(b []byte) uint64 {
	var num uint64
	buff := bytes.NewReader(b)
	err := binary.Read(buff, binary.BigEndian, &num)
	if err != nil {
		panic(err)
	}
	return num
}

// BigIntToBytes documents the QuantumCoin component used by this module.
// BigIntToBytes: *big.Int → []byte
func BigIntToBytes(n *big.Int) []byte {
	return n.Bytes()
}
