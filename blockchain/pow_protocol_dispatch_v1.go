package blockchain

import (
	"bytes"
	"fmt"
	"math/big"

	qconsensus "quantumcoin/consensus"
)

// ValidateBlockPoWV4 verifies the stored V4 hash and its proof-of-work target.
func ValidateBlockPoWV4(block *Block) error {
	if block == nil {
		return fmt.Errorf("canonical PoW V4: block cannot be nil")
	}
	if len(block.Hash) != 32 {
		return fmt.Errorf("canonical PoW V4: stored hash must be 32 bytes")
	}

	header, err := canonicalPoWHeaderV4(block)
	if err != nil {
		return err
	}

	hash, err := qconsensus.ComputePoWHashV4(header, uint64(block.Nonce))
	if err != nil {
		return fmt.Errorf("canonical PoW V4: compute hash: %w", err)
	}
	if !bytes.Equal(block.Hash, hash[:]) {
		return fmt.Errorf("canonical PoW V4: stored hash does not match header")
	}

	var hashValue big.Int
	hashValue.SetBytes(hash[:])
	target := qconsensus.TargetFromDifficultyV1(block.Difficulty)
	if hashValue.Cmp(target) >= 0 {
		return fmt.Errorf("canonical PoW V4: hash does not meet target")
	}

	return nil
}

// ValidateBlockPoWAtProtocolV1 dispatches PoW validation by the already
// selected consensus protocol. The caller remains responsible for selecting
// that protocol from the authoritative activation policy and block height.
func ValidateBlockPoWAtProtocolV1(
	block *Block,
	protocol qconsensus.ProtocolVersion,
) error {
	switch protocol {
	case qconsensus.ProtocolLegacyV1,
		qconsensus.ProtocolContinuityV2,
		qconsensus.ProtocolAtomV3:
		return ValidateBlockPoWV1(block)

	case qconsensus.ProtocolNFTV4:
		return ValidateBlockPoWV4(block)

	default:
		return fmt.Errorf("canonical PoW: unsupported protocol %q", protocol)
	}
}
