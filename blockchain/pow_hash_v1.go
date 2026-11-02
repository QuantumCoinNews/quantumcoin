package blockchain

import (
	"fmt"
	"strconv"

	qconsensus "quantumcoin/consensus"
)

const QCCanonicalPoWHashVersionV1 = qconsensus.CanonicalPoWHashVersionV1

// MaxConsensusNonceV1 is preserved in the blockchain package for
// compatibility. The authoritative limit is defined by consensus.
const MaxConsensusNonceV1 uint64 = qconsensus.MaxConsensusNonceV1

// canonicalPoWHeaderV1 converts the historical blockchain Block model into
// the chain-independent canonical consensus PoW header.
func canonicalPoWHeaderV1(
	block *Block,
) (qconsensus.PoWHeaderV1, error) {
	if block == nil {
		return qconsensus.PoWHeaderV1{}, fmt.Errorf(
			"%s: block cannot be nil",
			QCCanonicalPoWHashVersionV1,
		)
	}

	return qconsensus.PoWHeaderV1{
		PreviousHash: append(
			[]byte(nil),
			block.PrevHash...,
		),
		TransactionDigest: append(
			[]byte(nil),
			block.HashTransactions()...,
		),
		Timestamp:  block.Timestamp,
		Difficulty: block.Difficulty,
	}, nil
}

// SerializePoWHeaderV1 preserves the historical blockchain API while
// delegating canonical byte serialization to the consensus package.
func SerializePoWHeaderV1(
	block *Block,
	nonce int,
) ([]byte, error) {
	if nonce < 0 {
		return nil, fmt.Errorf(
			"%s: nonce cannot be negative: %d",
			QCCanonicalPoWHashVersionV1,
			nonce,
		)
	}

	header, err := canonicalPoWHeaderV1(block)
	if err != nil {
		return nil, err
	}

	return qconsensus.SerializePoWHeaderV1(
		header,
		uint64(nonce),
	)
}

// ComputePoWHashV1 preserves the historical blockchain API while delegating
// canonical SHA-256 computation to the consensus package.
func ComputePoWHashV1(
	block *Block,
	nonce int,
) ([32]byte, error) {
	var zero [32]byte

	if nonce < 0 {
		return zero, fmt.Errorf(
			"%s: nonce cannot be negative: %d",
			QCCanonicalPoWHashVersionV1,
			nonce,
		)
	}

	header, err := canonicalPoWHeaderV1(block)
	if err != nil {
		return zero, err
	}

	return qconsensus.ComputePoWHashV1(
		header,
		uint64(nonce),
	)
}

// PoWHashMeetsTargetV1 preserves the historical blockchain API while
// delegating canonical hash and target validation to the consensus package.
func PoWHashMeetsTargetV1(
	block *Block,
	nonce int,
) (bool, [32]byte, error) {
	var zero [32]byte

	if nonce < 0 {
		return false, zero, fmt.Errorf(
			"%s: nonce cannot be negative: %d",
			QCCanonicalPoWHashVersionV1,
			nonce,
		)
	}

	header, err := canonicalPoWHeaderV1(block)
	if err != nil {
		return false, zero, err
	}

	return qconsensus.PoWHashMeetsTargetV1(
		header,
		uint64(nonce),
	)
}

// ValidateBlockPoWV1 verifies the stored hash and target through the
// authoritative consensus implementation.
func ValidateBlockPoWV1(
	block *Block,
) error {
	header, err := canonicalPoWHeaderV1(block)
	if err != nil {
		return err
	}

	if block.Nonce < 0 {
		return fmt.Errorf(
			"%s: nonce cannot be negative: %d",
			QCCanonicalPoWHashVersionV1,
			block.Nonce,
		)
	}

	return qconsensus.ValidateStoredPoWHashV1(
		header,
		uint64(block.Nonce),
		block.Hash,
	)
}

// ConsensusNonceIntV1 converts the canonical uint64 mining nonce to the
// historical Block.Nonce representation without platform-dependent
// truncation.
func ConsensusNonceIntV1(
	nonce uint64,
) (int, error) {
	if strconv.IntSize != 64 {
		return 0, fmt.Errorf(
			"%s: a 64-bit runtime is required for consensus nonce storage",
			QCCanonicalPoWHashVersionV1,
		)
	}

	if nonce > MaxConsensusNonceV1 {
		return 0, fmt.Errorf(
			"%s: nonce %d exceeds consensus maximum %d",
			QCCanonicalPoWHashVersionV1,
			nonce,
			MaxConsensusNonceV1,
		)
	}

	return int(nonce), nil
}

// SerializePoWHeaderUint64V1 exposes the canonical fixed-width serializer to
// mining clients without an unchecked uint64-to-int conversion.
func SerializePoWHeaderUint64V1(
	block *Block,
	nonce uint64,
) ([]byte, error) {
	header, err := canonicalPoWHeaderV1(block)
	if err != nil {
		return nil, err
	}

	return qconsensus.SerializePoWHeaderV1(
		header,
		nonce,
	)
}

// ComputePoWHashUint64V1 computes the canonical PoW hash directly with a
// uint64 nonce.
func ComputePoWHashUint64V1(
	block *Block,
	nonce uint64,
) ([32]byte, error) {
	var zero [32]byte

	header, err := canonicalPoWHeaderV1(block)
	if err != nil {
		return zero, err
	}

	return qconsensus.ComputePoWHashV1(
		header,
		nonce,
	)
}

// PoWHashMeetsTargetUint64V1 performs the canonical mining-client target
// check directly with a uint64 nonce.
func PoWHashMeetsTargetUint64V1(
	block *Block,
	nonce uint64,
) (bool, [32]byte, error) {
	var zero [32]byte

	header, err := canonicalPoWHeaderV1(block)
	if err != nil {
		return false, zero, err
	}

	return qconsensus.PoWHashMeetsTargetV1(
		header,
		nonce,
	)
}
