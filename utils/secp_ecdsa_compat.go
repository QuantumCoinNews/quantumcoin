package utils

import (
	"bytes"
	"crypto/ecdsa"
	"errors"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// SecpPublicBytesV1 validates and serializes the legacy ECDSA secp256k1 facade.
// Go's PublicKey.Bytes only supports NIST curves, not secp256k1. Keep the
// unavoidable raw-field reads here; consumers never mutate coordinates.
func SecpPublicBytesV1(pub *ecdsa.PublicKey) ([]byte, error) {
	if pub == nil || pub.Curve != secp256k1.S256() {
		return nil, errors.New("secp256k1 public key required")
	}
	//lint:ignore SA1019 secp256k1 is unsupported by Go PublicKey.Bytes; validated compatibility boundary.
	x, y := pub.X, pub.Y
	if x == nil || y == nil || !secp256k1.S256().IsOnCurve(x, y) {
		return nil, errors.New("invalid secp256k1 public point")
	}
	raw := make([]byte, 65)
	raw[0] = 4
	x.FillBytes(raw[1:33])
	y.FillBytes(raw[33:65])
	parsed, err := secp256k1.ParsePubKey(raw)
	if err != nil {
		return nil, errors.New("invalid secp256k1 public encoding")
	}
	return parsed.SerializeUncompressed(), nil
}

// ImportRawSecpKeyV1 imports exactly one valid scalar without modular reduction.
func ImportRawSecpKeyV1(raw []byte) (*ecdsa.PrivateKey, error) {
	if len(raw) != 32 {
		return nil, errors.New("32-byte secp256k1 scalar required")
	}
	var scalar secp256k1.ModNScalar
	if scalar.SetByteSlice(raw) || scalar.IsZero() {
		scalar.Zero()
		return nil, errors.New("invalid secp256k1 scalar")
	}
	defer scalar.Zero()
	private := secp256k1.NewPrivateKey(&scalar)
	defer private.Zero()
	return private.ToECDSA(), nil
}

// SecpPrivateBytesV1 exports a validated scalar and checks its public identity.
func SecpPrivateBytesV1(private *ecdsa.PrivateKey) ([]byte, error) {
	if private == nil || private.Curve != secp256k1.S256() {
		return nil, errors.New("secp256k1 private key required")
	}
	//lint:ignore SA1019 secp256k1 is unsupported by Go PrivateKey.Bytes; bounded read only, never mutate a signing key.
	d := private.D
	if d == nil || d.Sign() <= 0 || d.BitLen() > 256 {
		return nil, errors.New("invalid secp256k1 scalar")
	}
	raw := make([]byte, 32)
	d.FillBytes(raw)
	imported, err := ImportRawSecpKeyV1(raw)
	if err != nil {
		clear(raw)
		return nil, err
	}
	defer ZeroSecpPrivateV1(imported)
	actual, err := SecpPublicBytesV1(&private.PublicKey)
	if err != nil {
		clear(raw)
		return nil, err
	}
	expected, err := SecpPublicBytesV1(&imported.PublicKey)
	if err != nil || !bytes.Equal(actual, expected) {
		clear(raw)
		return nil, errors.New("secp256k1 private/public key mismatch")
	}
	return raw, nil
}

// ZeroSecpPrivateV1 clears an exclusively owned, discarded compatibility key.
// It must not run concurrently with signing or be used as a reusable-key API.
func ZeroSecpPrivateV1(private *ecdsa.PrivateKey) {
	if private == nil || private.Curve != secp256k1.S256() {
		return
	}
	//lint:ignore SA1019 legacy secp256k1 facade cleanup after last use; no supported Go scalar cleanup API.
	d := private.D
	if d != nil {
		d.SetInt64(0)
	}
}
