package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
)

const (
	// CanonicalPoWHashVersionV1 mevcut zincirin tarihsel
	// PoW header ve hash sözleşmesidir.
	CanonicalPoWHashVersionV1 = "QC_CANONICAL_POW_HASH_V1"

	// PoWHashBitsV1 SHA-256 çıktı genişliğidir.
	PoWHashBitsV1 = 256

	// MaxConsensusNonceV1 mevcut Block.Nonce int alanında
	// 64-bit çalışma ortamında güvenle saklanabilen en büyük nonce'tur.
	MaxConsensusNonceV1 uint64 = 1<<63 - 1
)

// PoWHeaderV1 mevcut zincirin canonical PoW preimage alanlarını taşır.
//
// Index, saklanan Hash, Miner ve Metadata bu yapıya dahil değildir.
// Bu alanların eklenmesi eski blok hashlerini değiştirir.
type PoWHeaderV1 struct {
	PreviousHash      []byte
	TransactionDigest []byte
	Timestamp         int64
	Difficulty        int
}

// NormalizePoWDifficultyV1 mevcut legacy PoW difficulty değerini
// 0 ile 255 arasında sınırlar.
func NormalizePoWDifficultyV1(difficulty int) int {
	if difficulty < 0 {
		return 0
	}

	if difficulty > MaxDifficultyBits {
		return MaxDifficultyBits
	}

	return difficulty
}

// TargetFromDifficultyV1 mevcut zincirde kullanılan target hesabını yapar.
//
// target = 1 << (256 - difficulty)
func TargetFromDifficultyV1(difficulty int) *big.Int {
	difficulty = NormalizePoWDifficultyV1(difficulty)

	target := big.NewInt(1)
	shift := uint(PoWHashBitsV1 - difficulty)

	return target.Lsh(target, shift)
}

// SerializePoWHeaderV1 mevcut zincirin tarihsel PoW preimage
// byte düzenini üretir.
//
// Byte sırası:
//   - previous hash: ham byte
//   - transaction digest: 32 byte
//   - timestamp: signed int64, 8-byte big-endian
//   - difficulty: signed int64, 8-byte big-endian
//   - nonce: uint64, 8-byte big-endian
func SerializePoWHeaderV1(
	header PoWHeaderV1,
	nonce uint64,
) ([]byte, error) {
	if len(header.TransactionDigest) != sha256.Size {
		return nil, fmt.Errorf(
			"%s: transaction digest length is %d, expected %d",
			CanonicalPoWHashVersionV1,
			len(header.TransactionDigest),
			sha256.Size,
		)
	}

	if nonce > MaxConsensusNonceV1 {
		return nil, fmt.Errorf(
			"%s: nonce %d exceeds consensus maximum %d",
			CanonicalPoWHashVersionV1,
			nonce,
			MaxConsensusNonceV1,
		)
	}

	data := make(
		[]byte,
		0,
		len(header.PreviousHash)+sha256.Size+(3*8),
	)

	data = append(data, header.PreviousHash...)
	data = append(data, header.TransactionDigest...)

	data = appendInt64BigEndianV1(
		data,
		header.Timestamp,
	)

	data = appendInt64BigEndianV1(
		data,
		int64(header.Difficulty),
	)

	data = appendUint64BigEndianV1(
		data,
		nonce,
	)

	return data, nil
}

// ComputePoWHashV1 canonical PoW header üzerinde tek tur SHA-256 uygular.
func ComputePoWHashV1(
	header PoWHeaderV1,
	nonce uint64,
) ([32]byte, error) {
	var zero [32]byte

	data, err := SerializePoWHeaderV1(
		header,
		nonce,
	)
	if err != nil {
		return zero, err
	}

	return sha256.Sum256(data), nil
}

// PoWHashMeetsTargetV1 canonical PoW hashinin target değerinden
// kesin olarak küçük olup olmadığını kontrol eder.
func PoWHashMeetsTargetV1(
	header PoWHeaderV1,
	nonce uint64,
) (bool, [32]byte, error) {
	var zero [32]byte

	hash, err := ComputePoWHashV1(
		header,
		nonce,
	)
	if err != nil {
		return false, zero, err
	}

	target := TargetFromDifficultyV1(
		header.Difficulty,
	)

	var hashInteger big.Int
	hashInteger.SetBytes(hash[:])

	return hashInteger.Cmp(target) == -1, hash, nil
}

// ValidateStoredPoWHashV1 saklanan hash ile canonical hesaplanan hashin
// aynı olduğunu ve target şartını karşıladığını doğrular.
func ValidateStoredPoWHashV1(
	header PoWHeaderV1,
	nonce uint64,
	storedHash []byte,
) error {
	if len(storedHash) != sha256.Size {
		return fmt.Errorf(
			"%s: stored block hash length is %d, expected %d",
			CanonicalPoWHashVersionV1,
			len(storedHash),
			sha256.Size,
		)
	}

	meetsTarget, computedHash, err := PoWHashMeetsTargetV1(
		header,
		nonce,
	)
	if err != nil {
		return err
	}

	if !equalHashV1(storedHash, computedHash[:]) {
		return fmt.Errorf(
			"%s: stored block hash does not match canonical PoW hash",
			CanonicalPoWHashVersionV1,
		)
	}

	if !meetsTarget {
		return fmt.Errorf(
			"%s: canonical PoW hash does not meet target",
			CanonicalPoWHashVersionV1,
		)
	}

	return nil
}

func appendInt64BigEndianV1(
	destination []byte,
	value int64,
) []byte {
	return appendUint64BigEndianV1(
		destination,
		uint64(value),
	)
}

func appendUint64BigEndianV1(
	destination []byte,
	value uint64,
) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)

	return append(destination, encoded[:]...)
}

func equalHashV1(
	left []byte,
	right []byte,
) bool {
	if len(left) != len(right) {
		return false
	}

	var difference byte

	for index := range left {
		difference |= left[index] ^ right[index]
	}

	return difference == 0
}
