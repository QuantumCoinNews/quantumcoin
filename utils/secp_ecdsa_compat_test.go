package utils

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestSecpCompatibilityPreservesScalarAndKnownPublicPoint(t *testing.T) {
	raw := make([]byte, 32)
	raw[31] = 1
	key, err := ImportRawSecpKeyV1(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer ZeroSecpPrivateV1(key)
	if _, err := key.Bytes(); err == nil {
		t.Fatal("review required: Go now supports secp256k1 Bytes; remove compatibility exception")
	}
	public, err := SecpPublicBytesV1(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(public) != "0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8" {
		t.Fatal("secp256k1 public encoding changed")
	}
	exported, err := SecpPrivateBytesV1(key)
	if err != nil || !bytes.Equal(exported, raw) {
		t.Fatal("scalar roundtrip changed")
	}
	clear(exported)
	public[1] ^= 1
	second, err := SecpPublicBytesV1(&key.PublicKey)
	if err != nil || bytes.Equal(public, second) {
		t.Fatal("public export aliases key")
	}
	ZeroSecpPrivateV1(key)
	if _, err := SecpPrivateBytesV1(key); err == nil {
		t.Fatal("cleared key exported")
	}
}

func TestSecpCompatibilityRejectsInvalidAndMismatchedKeys(t *testing.T) {
	for _, raw := range [][]byte{nil, make([]byte, 31), make([]byte, 32), bytes.Repeat([]byte{255}, 32)} {
		if _, err := ImportRawSecpKeyV1(raw); err == nil {
			t.Fatal("invalid scalar accepted")
		}
	}
	a, b := make([]byte, 32), make([]byte, 32)
	a[31] = 1
	b[31] = 2
	first, _ := ImportRawSecpKeyV1(a)
	second, _ := ImportRawSecpKeyV1(b)
	defer ZeroSecpPrivateV1(first)
	defer ZeroSecpPrivateV1(second)
	first.PublicKey = second.PublicKey
	if _, err := SecpPrivateBytesV1(first); err == nil {
		t.Fatal("mismatched key accepted")
	}
	if _, err := SecpPublicBytesV1(nil); err == nil {
		t.Fatal("nil public key accepted")
	}
}
