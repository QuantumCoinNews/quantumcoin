package blockchain

import (
	"fmt"
)

// PEER_BLOCK_VALIDATION_V2
//
// This file hardens blocks received from peers.
// It does not mint, transfer, change rewards, or change consensus policy by itself.
// It only rejects unsafe peer-supplied blocks before they are appended.

const MinPeerBlockDifficultyV2 = 1

// AddBlockFromPeerStrictV2 validates a peer block with strict checks and then
// delegates to the existing AddBlockFromPeer append path.
func (bc *Blockchain) AddBlockFromPeerStrictV2(blk *Block) error {
	if err := bc.ValidatePeerBlockStrictV2(blk); err != nil {
		logPeerBlockReject(
			"BLOCK_PEER_STRICT_VALIDATION_FAILED",
			blk,
			err,
		)
		return err
	}

	return bc.addBlockFromPeerAfterStrictV2(blk)
}

// ValidatePeerBlockStrictV2 performs all strict checks that are safe to run
// before accepting one next block from a peer.
func (bc *Blockchain) ValidatePeerBlockStrictV2(blk *Block) error {
	if bc == nil {
		return fmt.Errorf(
			"peer block strict validation: nil blockchain",
		)
	}

	if len(bc.Blocks) == 0 {
		return ErrEmptyChain
	}

	if err := bc.ValidateNewBlockBasic(blk); err != nil {
		return fmt.Errorf(
			"peer block strict basic validation failed: %w",
			err,
		)
	}

	protocol := bc.ProtocolAtHeightV1(
		blk.Index,
	)
	if err := validateNetworkBlockProtocolV1(
		protocol,
	); err != nil {
		return fmt.Errorf(
			"peer block strict validation: %w",
			err,
		)
	}

	if blk.Difficulty < MinPeerBlockDifficultyV2 ||
		blk.Difficulty > maxBits {
		return fmt.Errorf(
			"peer block invalid difficulty: %d",
			blk.Difficulty,
		)
	}

	if err := bc.ValidateExpectedDifficultyV1(blk); err != nil {
		return fmt.Errorf(
			"peer block expected difficulty validation failed: %w",
			err,
		)
	}
	if _, err := bc.ValidateCanonicalBlockEnvelopeV1(
		blk,
		bc.TotalMinted(),
	); err != nil {
		return fmt.Errorf(
			"peer block canonical consensus validation failed: %w",
			err,
		)
	}

	if err := bc.ValidateFounderVestingBlockV1(blk); err != nil {
		return fmt.Errorf(
			"peer block founder vesting validation failed: %w",
			err,
		)
	}

	if err := bc.validatePeerBlockSpendsV2(blk); err != nil {
		return err
	}

	return nil
}

func (bc *Blockchain) validatePeerBlockSpendsV2(
	blk *Block,
) error {
	if bc == nil {
		return fmt.Errorf(
			"nil blockchain",
		)
	}

	if blk == nil {
		return ErrNilBlock
	}

	if _, err := bc.ValidateTransactionsUTXOAtHeightV1(
		blk.Transactions,
		blk.Index,
	); err != nil {
		return fmt.Errorf(
			"peer block canonical UTXO validation failed: %w",
			err,
		)
	}

	return nil
}

func (bc *Blockchain) AddBlockFromPeer(
	blk *Block,
) error {
	return bc.AddBlockFromPeerStrictV2(
		blk,
	)
}
