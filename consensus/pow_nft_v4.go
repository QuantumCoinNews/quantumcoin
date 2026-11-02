package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
)

const canonicalPoWHashVersionV4 = "QC_CANONICAL_POW_HASH_V4"
const powHeaderV4Domain = "QC_POW_HEADER_V4\x00"

// PoWHeaderV4 is the canonical V4 header. NFTRecordDigest commits the
// block's canonical NFT mint-record set into proof of work.
type PoWHeaderV4 struct {
	PreviousHash      []byte
	TransactionDigest []byte
	NFTRecordDigest   []byte
	Timestamp         int64
	Difficulty        int
}

// SerializePoWHeaderV4 returns a fixed-width, domain-separated header encoding.
// V1 serialization remains unchanged for historical blocks.
func SerializePoWHeaderV4(header PoWHeaderV4, nonce uint64) ([]byte, error) {
	if len(header.PreviousHash) != sha256.Size {
		return nil, fmt.Errorf(
			"%s: previous hash length is %d, expected %d",
			canonicalPoWHashVersionV4,
			len(header.PreviousHash),
			sha256.Size,
		)
	}
	if len(header.TransactionDigest) != sha256.Size {
		return nil, fmt.Errorf(
			"%s: transaction digest length is %d, expected %d",
			canonicalPoWHashVersionV4,
			len(header.TransactionDigest),
			sha256.Size,
		)
	}
	if len(header.NFTRecordDigest) != sha256.Size {
		return nil, fmt.Errorf(
			"%s: NFT record digest length is %d, expected %d",
			canonicalPoWHashVersionV4,
			len(header.NFTRecordDigest),
			sha256.Size,
		)
	}
	if header.Timestamp < 0 {
		return nil, fmt.Errorf(
			"%s: timestamp cannot be negative",
			canonicalPoWHashVersionV4,
		)
	}
	if header.Difficulty < 0 || header.Difficulty > MaxDifficultyBits {
		return nil, fmt.Errorf(
			"%s: difficulty %d is outside 0..%d",
			canonicalPoWHashVersionV4,
			header.Difficulty,
			MaxDifficultyBits,
		)
	}
	if nonce > MaxConsensusNonceV1 {
		return nil, fmt.Errorf(
			"%s: nonce %d exceeds consensus maximum %d",
			canonicalPoWHashVersionV4,
			nonce,
			MaxConsensusNonceV1,
		)
	}

	data := make([]byte, 0, len(powHeaderV4Domain)+(3*sha256.Size)+(3*8))
	data = append(data, powHeaderV4Domain...)
	data = append(data, header.PreviousHash...)
	data = append(data, header.TransactionDigest...)
	data = append(data, header.NFTRecordDigest...)

	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], uint64(header.Timestamp))
	data = append(data, encoded[:]...)
	binary.BigEndian.PutUint64(encoded[:], uint64(header.Difficulty))
	data = append(data, encoded[:]...)
	binary.BigEndian.PutUint64(encoded[:], nonce)
	data = append(data, encoded[:]...)

	return data, nil
}

// ComputePoWHashV4 returns the SHA-256 hash of the canonical V4 header bytes.
func ComputePoWHashV4(header PoWHeaderV4, nonce uint64) ([32]byte, error) {
	var zero [32]byte

	data, err := SerializePoWHeaderV4(header, nonce)
	if err != nil {
		return zero, err
	}
	return sha256.Sum256(data), nil
}

func PoWHashMeetsTargetV4(
	header PoWHeaderV4,
	nonce uint64,
) (bool, [32]byte, error) {
	var zero [32]byte

	hash, err := ComputePoWHashV4(header, nonce)
	if err != nil {
		return false, zero, err
	}

	target := TargetFromDifficultyV1(header.Difficulty)

	var hashInteger big.Int
	hashInteger.SetBytes(hash[:])

	return hashInteger.Cmp(target) == -1, hash, nil
}

func ValidateStoredPoWHashV4(
	header PoWHeaderV4,
	nonce uint64,
	storedHash []byte,
) error {
	if len(storedHash) != sha256.Size {
		return fmt.Errorf(
			"%s: stored block hash length is %d, expected %d",
			canonicalPoWHashVersionV4,
			len(storedHash),
			sha256.Size,
		)
	}

	meetsTarget, computedHash, err := PoWHashMeetsTargetV4(
		header,
		nonce,
	)
	if err != nil {
		return err
	}

	if !equalHashV1(storedHash, computedHash[:]) {
		return fmt.Errorf(
			"%s: stored block hash does not match canonical PoW hash",
			canonicalPoWHashVersionV4,
		)
	}

	if !meetsTarget {
		return fmt.Errorf(
			"%s: canonical PoW hash does not meet target",
			canonicalPoWHashVersionV4,
		)
	}

	return nil
}
