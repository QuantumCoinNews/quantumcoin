package utils

import (
	"bytes"
	"encoding/gob"
)

// EncodeToBytes: Herhangi bir nesneyi gob ile []byte'a çevirir
func EncodeToBytes(data interface{}) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// DecodeFromBytes: Gob []byte'dan istenen nesneyi üretir
func DecodeFromBytes(data []byte, target interface{}) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(target)
}
