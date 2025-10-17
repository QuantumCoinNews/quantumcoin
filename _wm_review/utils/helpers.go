package utils

import (
	"bytes"
	"encoding/gob"
)

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

// DecodeFromBytes: Gob decode []byte to struct
func DecodeFromBytes(data []byte, target interface{}) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(target)
}
