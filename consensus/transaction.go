package consensus

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

const (
	// TransactionConsensusVersionV1 mevcut zincirin transaction ID,
	// imza özeti ve canonical imza sözleşmesidir.
	TransactionConsensusVersionV1 = "QC_TRANSACTION_ID_SIGNATURE_V1"
)

// CanonicalTransactionIDFromPayloadV1 blockchain katmanının mevcut
// Transaction.TrimmedCopy + gob serialization yöntemiyle ürettiği
// kanonik payload üzerinde tek tur SHA-256 uygular.
//
// Payload'ın consensus paketi dışında hazırlanmasının nedeni,
// tarihsel gob byte düzenini ve eski transaction ID değerlerini korumaktır.
func CanonicalTransactionIDFromPayloadV1(
	payload []byte,
) ([32]byte, error) {
	var zero [32]byte

	if len(payload) == 0 {
		return zero, fmt.Errorf(
			"%s: transaction ID payload is empty",
			TransactionConsensusVersionV1,
		)
	}

	return sha256.Sum256(payload), nil
}

// ValidateTransactionIDFromPayloadV1 saklanan transaction ID'nin,
// kanonik payload üzerinden yeniden hesaplanan ID ile aynı olduğunu doğrular.
func ValidateTransactionIDFromPayloadV1(
	storedID []byte,
	payload []byte,
) error {
	if len(storedID) != sha256.Size {
		return fmt.Errorf(
			"%s: transaction ID length is %d, want %d",
			TransactionConsensusVersionV1,
			len(storedID),
			sha256.Size,
		)
	}

	expectedID, err :=
		CanonicalTransactionIDFromPayloadV1(payload)
	if err != nil {
		return err
	}

	if !bytes.Equal(storedID, expectedID[:]) {
		return fmt.Errorf(
			"%s: transaction ID does not match canonical content",
			TransactionConsensusVersionV1,
		)
	}

	return nil
}

// SigningDigestV1 mevcut transaction input imza özetini üretir.
//
// Tarihsel byte düzeni:
//
//	trimmed transaction gob serialization
//	input transaction ID raw bytes
//	output index uint32 big-endian
//
// Mevcut zincir davranışını korumak için outIndex değeri uint32
// dönüşümüyle dört byte olarak yazılır.
func SigningDigestV1(
	trimmedSerializedTransaction []byte,
	inputTransactionID []byte,
	outIndex int,
) ([32]byte, error) {
	var zero [32]byte

	if len(trimmedSerializedTransaction) == 0 {
		return zero, fmt.Errorf(
			"%s: trimmed transaction payload is empty",
			TransactionConsensusVersionV1,
		)
	}

	payload := make(
		[]byte,
		0,
		len(trimmedSerializedTransaction)+
			len(inputTransactionID)+4,
	)

	payload = append(
		payload,
		trimmedSerializedTransaction...,
	)

	payload = append(
		payload,
		inputTransactionID...,
	)

	var encodedOutputIndex [4]byte

	binary.BigEndian.PutUint32(
		encodedOutputIndex[:],
		uint32(outIndex),
	)

	payload = append(
		payload,
		encodedOutputIndex[:]...,
	)

	return sha256.Sum256(payload), nil
}

// NormalizeTransactionLowSV1 secp256k1 imzasındaki S değerini
// canonical low-S biçimine dönüştürür.
func NormalizeTransactionLowSV1(
	s *big.Int,
) *big.Int {
	if s == nil {
		return nil
	}

	normalized := new(big.Int).Set(s)
	halfOrder := transactionHalfCurveOrderV1()

	if normalized.Cmp(halfOrder) > 0 {
		normalized.Sub(
			transactionCurveOrderV1(),
			normalized,
		)
	}

	return normalized
}

// EncodeCanonicalTransactionSignatureV1 r ve s değerlerini
// mevcut QuantumCoin signature byte biçiminde kodlar.
//
// Düzen:
//
//	r uzunluğu: 1 byte
//	r: 1–32 byte
//	s uzunluğu: 1 byte
//	s: 1–32 byte
func EncodeCanonicalTransactionSignatureV1(
	r *big.Int,
	s *big.Int,
) ([]byte, error) {
	if r == nil || s == nil {
		return nil, fmt.Errorf(
			"%s: signature values cannot be nil",
			TransactionConsensusVersionV1,
		)
	}

	order := transactionCurveOrderV1()

	if r.Sign() <= 0 || r.Cmp(order) >= 0 {
		return nil, fmt.Errorf(
			"%s: r is outside secp256k1 range",
			TransactionConsensusVersionV1,
		)
	}

	normalizedS := NormalizeTransactionLowSV1(s)

	if normalizedS == nil ||
		normalizedS.Sign() <= 0 ||
		normalizedS.Cmp(order) >= 0 {
		return nil, fmt.Errorf(
			"%s: s is outside secp256k1 range",
			TransactionConsensusVersionV1,
		)
	}

	rBytes := r.Bytes()
	sBytes := normalizedS.Bytes()

	if len(rBytes) < 1 || len(rBytes) > 32 {
		return nil, fmt.Errorf(
			"%s: invalid r length %d",
			TransactionConsensusVersionV1,
			len(rBytes),
		)
	}

	if len(sBytes) < 1 || len(sBytes) > 32 {
		return nil, fmt.Errorf(
			"%s: invalid s length %d",
			TransactionConsensusVersionV1,
			len(sBytes),
		)
	}

	encoded := make(
		[]byte,
		0,
		1+len(rBytes)+1+len(sBytes),
	)

	encoded = append(
		encoded,
		byte(len(rBytes)),
	)

	encoded = append(
		encoded,
		rBytes...,
	)

	encoded = append(
		encoded,
		byte(len(sBytes)),
	)

	encoded = append(
		encoded,
		sBytes...,
	)

	return encoded, nil
}

// DecodeCanonicalTransactionSignatureV1 yalnızca minimal,
// aralık içinde ve low-S olan canonical imzaları kabul eder.
func DecodeCanonicalTransactionSignatureV1(
	signature []byte,
) (*big.Int, *big.Int, error) {
	if len(signature) < 4 {
		return nil, nil, fmt.Errorf(
			"%s: signature is too short",
			TransactionConsensusVersionV1,
		)
	}

	rLength := int(signature[0])

	if rLength < 1 || rLength > 32 {
		return nil, nil, fmt.Errorf(
			"%s: invalid r length %d",
			TransactionConsensusVersionV1,
			rLength,
		)
	}

	sLengthOffset := 1 + rLength

	if sLengthOffset >= len(signature) {
		return nil, nil, fmt.Errorf(
			"%s: signature is truncated before s length",
			TransactionConsensusVersionV1,
		)
	}

	sLength := int(signature[sLengthOffset])

	if sLength < 1 || sLength > 32 {
		return nil, nil, fmt.Errorf(
			"%s: invalid s length %d",
			TransactionConsensusVersionV1,
			sLength,
		)
	}

	expectedLength :=
		1 +
			rLength +
			1 +
			sLength

	if len(signature) != expectedLength {
		return nil, nil, fmt.Errorf(
			"%s: signature length is %d, want %d",
			TransactionConsensusVersionV1,
			len(signature),
			expectedLength,
		)
	}

	rBytes := signature[1 : 1+rLength]
	sBytes := signature[sLengthOffset+1:]

	if !isMinimalPositiveIntegerEncodingV1(rBytes) {
		return nil, nil, fmt.Errorf(
			"%s: r is not minimally encoded",
			TransactionConsensusVersionV1,
		)
	}

	if !isMinimalPositiveIntegerEncodingV1(sBytes) {
		return nil, nil, fmt.Errorf(
			"%s: s is not minimally encoded",
			TransactionConsensusVersionV1,
		)
	}

	r := new(big.Int).SetBytes(rBytes)
	s := new(big.Int).SetBytes(sBytes)
	order := transactionCurveOrderV1()

	if r.Sign() <= 0 || r.Cmp(order) >= 0 {
		return nil, nil, fmt.Errorf(
			"%s: r is outside secp256k1 range",
			TransactionConsensusVersionV1,
		)
	}

	if s.Sign() <= 0 || s.Cmp(order) >= 0 {
		return nil, nil, fmt.Errorf(
			"%s: s is outside secp256k1 range",
			TransactionConsensusVersionV1,
		)
	}

	if s.Cmp(transactionHalfCurveOrderV1()) > 0 {
		return nil, nil, fmt.Errorf(
			"%s: high-S signature is not canonical",
			TransactionConsensusVersionV1,
		)
	}

	canonicalSignature, err :=
		EncodeCanonicalTransactionSignatureV1(r, s)
	if err != nil {
		return nil, nil, err
	}

	if !bytes.Equal(signature, canonicalSignature) {
		return nil, nil, fmt.Errorf(
			"%s: signature encoding is not canonical",
			TransactionConsensusVersionV1,
		)
	}

	return r, s, nil
}

// ParseUncompressedPublicKeyV1 65 byte 0x04 || X || Y biçimindeki
// secp256k1 public key değerini doğrular.
func ParseUncompressedPublicKeyV1(
	publicKey []byte,
) (*ecdsa.PublicKey, error) {
	if len(publicKey) != 65 {
		return nil, fmt.Errorf(
			"%s: public key length is %d, want 65",
			TransactionConsensusVersionV1,
			len(publicKey),
		)
	}

	if publicKey[0] != 0x04 {
		return nil, fmt.Errorf(
			"%s: public key is not uncompressed",
			TransactionConsensusVersionV1,
		)
	}

	curve := secp256k1.S256()

	x := new(big.Int).SetBytes(
		publicKey[1:33],
	)

	y := new(big.Int).SetBytes(
		publicKey[33:65],
	)

	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf(
			"%s: public key is not on secp256k1 curve",
			TransactionConsensusVersionV1,
		)
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}

// UncompressedPublicKeyV1 private key içindeki public keyi
// mevcut transaction formatındaki 65 byte biçimde döndürür.
func UncompressedPublicKeyV1(
	privateKey *ecdsa.PrivateKey,
) ([]byte, error) {
	if privateKey == nil ||
		privateKey.PublicKey.X == nil ||
		privateKey.PublicKey.Y == nil {
		return nil, fmt.Errorf(
			"%s: private key is unavailable",
			TransactionConsensusVersionV1,
		)
	}

	if !secp256k1.S256().IsOnCurve(
		privateKey.PublicKey.X,
		privateKey.PublicKey.Y,
	) {
		return nil, fmt.Errorf(
			"%s: private key public point is not on secp256k1",
			TransactionConsensusVersionV1,
		)
	}

	publicKey := make(
		[]byte,
		0,
		65,
	)

	publicKey = append(publicKey, 0x04)

	publicKey = append(
		publicKey,
		padTransactionCoordinateV1(
			privateKey.PublicKey.X.Bytes(),
		)...,
	)

	publicKey = append(
		publicKey,
		padTransactionCoordinateV1(
			privateKey.PublicKey.Y.Bytes(),
		)...,
	)

	return publicKey, nil
}

// SignTransactionDigestV1 32 byte transaction input özetini
// secp256k1 ile imzalar ve canonical low-S biçiminde kodlar.
func SignTransactionDigestV1(
	privateKey *ecdsa.PrivateKey,
	digest [32]byte,
) ([]byte, error) {
	if privateKey == nil {
		return nil, fmt.Errorf(
			"%s: private key is nil",
			TransactionConsensusVersionV1,
		)
	}

	r, s, err := ecdsa.Sign(
		rand.Reader,
		privateKey,
		digest[:],
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: sign digest: %w",
			TransactionConsensusVersionV1,
			err,
		)
	}

	return EncodeCanonicalTransactionSignatureV1(
		r,
		s,
	)
}

// VerifyTransactionDigestSignatureV1 public key, canonical signature
// ve 32 byte digest üçlüsünü doğrular.
func VerifyTransactionDigestSignatureV1(
	publicKey []byte,
	digest [32]byte,
	signature []byte,
) error {
	parsedPublicKey, err :=
		ParseUncompressedPublicKeyV1(publicKey)
	if err != nil {
		return err
	}

	r, s, err :=
		DecodeCanonicalTransactionSignatureV1(signature)
	if err != nil {
		return err
	}

	if !ecdsa.Verify(
		parsedPublicKey,
		digest[:],
		r,
		s,
	) {
		return fmt.Errorf(
			"%s: signature verification failed",
			TransactionConsensusVersionV1,
		)
	}

	return nil
}

func transactionCurveOrderV1() *big.Int {
	return new(big.Int).Set(
		secp256k1.S256().Params().N,
	)
}

func transactionHalfCurveOrderV1() *big.Int {
	return new(big.Int).Rsh(
		transactionCurveOrderV1(),
		1,
	)
}

func isMinimalPositiveIntegerEncodingV1(
	encoded []byte,
) bool {
	if len(encoded) == 0 {
		return false
	}

	if len(encoded) > 1 && encoded[0] == 0 {
		return false
	}

	return true
}

func padTransactionCoordinateV1(
	value []byte,
) []byte {
	if len(value) >= 32 {
		return append([]byte(nil), value...)
	}

	padded := make([]byte, 32)

	copy(
		padded[32-len(value):],
		value,
	)

	return padded
}
