// Package utils contains QuantumCoin project code for this module.
package utils

import (
	"bytes"
	"encoding/gob"
)

// EncodeToBytes documents the QuantumCoin component used by this module.
// EncodeToBytes: Gob encode any data to []byte
func EncodeToBytes(data interface{}) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// DecodeFromBytes documents the QuantumCoin component used by this module.
// DecodeFromBytes: Gob decode []byte to struct
func DecodeFromBytes(data []byte, target interface{}) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(target)
}
