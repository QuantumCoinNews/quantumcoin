package utils

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/ripemd160"
)

// 32-byte left-pad
func Pad32(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// Uncompressed pub: 0x04 || X || Y  (65B)
func MakeUncompressedPub(priv *ecdsa.PrivateKey) []byte {
	pub := append([]byte{0x04}, Pad32(priv.PublicKey.X.Bytes())...)
	pub = append(pub, Pad32(priv.PublicKey.Y.Bytes())...)
	return pub
}

// PubKey (uncompressed) -> *ecdsa.PublicKey (secp256k1)
func PubKeyToECDSA(uncompressed []byte) *ecdsa.PublicKey {
	if len(uncompressed) != 65 || uncompressed[0] != 0x04 {
		return nil
	}
	curve := secp256k1.S256()
	x := new(big.Int).SetBytes(uncompressed[1:33])
	y := new(big.Int).SetBytes(uncompressed[33:65])
	if !curve.IsOnCurve(x, y) {
		return nil
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
}

// SHA256(msg) üstüne ECDSA imza, (r,s) döner
func SignMessage(priv *ecdsa.PrivateKey, msg []byte) (r, s *big.Int, err error) {
	h := sha256.Sum256(msg)
	return ecdsa.Sign(rand.Reader, priv, h[:])
}

// Doğrulama: uncompressed pub ile (r,s)
func VerifyMessage(uncompressed []byte, msg []byte, r, s *big.Int) bool {
	pub := PubKeyToECDSA(uncompressed)
	if pub == nil {
		return false
	}
	h := sha256.Sum256(msg)
	return ecdsa.Verify(pub, h[:], r, s)
}

// Hash160(pub) -> Base58Check adres (version=0x00)
func PubKeyToAddress(uncompressed []byte) string {
	sha := sha256.Sum256(uncompressed)
	r := ripemd160.New()
	_, _ = r.Write(sha[:])
	pkh := r.Sum(nil)

	version := byte(0x00)
	payload := append([]byte{version}, pkh...)
	cs := sha256.Sum256(payload)
	cs = sha256.Sum256(cs[:])
	full := append(payload, cs[:4]...)

	// Base58Encode []byte döndürüyor → string'e çevirelim
	return string(Base58Encode(full))
}

// Hex yardımcıları (isteğe bağlı)
func BytesToHex(b []byte) string          { return hex.EncodeToString(b) }
func HexToBytes(s string) ([]byte, error) { return hex.DecodeString(s) }
