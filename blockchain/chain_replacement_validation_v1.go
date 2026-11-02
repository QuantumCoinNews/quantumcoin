package blockchain

import (
	"bytes"
	"fmt"
)

// REPLACECHAIN_STRICT_VALIDATION_V1
//
// This file hardens longer-chain replacement from peers.
// It does not mint QC, change rewards, or select a chain by AI.
// It only rejects unsafe peer-supplied replacement chains before ReplaceChain.

// ReplaceChainFromPeerStrictV1 validates a peer-provided longer chain with
// strict checks, then delegates to the canonical ReplaceChain path.
func (bc *Blockchain) ReplaceChainFromPeerStrictV1(
	blocks []*Block,
) error {
	if err := bc.ValidateReplacementChainStrictV1(
		blocks,
	); err != nil {
		return err
	}

	return bc.replaceChainAfterStrictV1(
		blocks,
	)
}

// ValidateReplacementChainStrictV1 validates a candidate replacement chain
// without mutating the current chain.
func (bc *Blockchain) ValidateReplacementChainStrictV1(
	blocks []*Block,
) error {
	if bc == nil {
		return fmt.Errorf(
			"replacechain strict validation: nil blockchain",
		)
	}

	if len(bc.Blocks) == 0 {
		return ErrEmptyChain
	}

	if len(blocks) == 0 {
		return ErrEmptyChain
	}

	if len(blocks) <= len(bc.Blocks) {
		return ErrIncomingChainNotLonger
	}
	return bc.validateCanonicalChainWithLocalContextV1(blocks)
}

// validateCanonicalChainWithLocalContextV1 replays an entire chain using
// trusted local genesis/economics/runtime policies, without selecting it as
// a longer peer branch. Callers hold the source lock or use a detached snapshot.
func (bc *Blockchain) validateCanonicalChainWithLocalContextV1(blocks []*Block) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	if len(bc.Blocks) == 0 || len(blocks) == 0 {
		return ErrEmptyChain
	}
	if blocks[0] == nil {
		return fmt.Errorf(
			"incoming chain has nil genesis block",
		)
	}

	if bc.Blocks[0] == nil {
		return fmt.Errorf(
			"local chain has nil genesis block",
		)
	}

	if err := ValidateBlockBasic(
		blocks[0],
	); err != nil {
		return fmt.Errorf(
			"incoming chain genesis basic validation failed: %w",
			err,
		)
	}

	if err := validateNFTMintAuthorizationV1(
		blocks[0],
		bc.ProtocolAtHeightV1(blocks[0].Index),
	); err != nil {
		return fmt.Errorf(
			"incoming chain genesis NFT mint authorization failed: %w",
			err,
		)
	}

	localGenesisContentHash, err := ComputePoWHashV1(
		bc.Blocks[0],
		bc.Blocks[0].Nonce,
	)
	if err != nil {
		return fmt.Errorf(
			"local chain genesis content hash failed: %w",
			err,
		)
	}

	incomingGenesisContentHash, err := ComputePoWHashV1(
		blocks[0],
		blocks[0].Nonce,
	)
	if err != nil {
		return fmt.Errorf(
			"incoming chain genesis content hash failed: %w",
			err,
		)
	}

	if !bytes.Equal(
		incomingGenesisContentHash[:],
		localGenesisContentHash[:],
	) {
		return fmt.Errorf(
			"incoming chain genesis content hash mismatch",
		)
	}

	if blocks[0].Miner != bc.Blocks[0].Miner {
		return fmt.Errorf(
			"incoming chain genesis miner mismatch",
		)
	}

	if len(blocks[0].Metadata) !=
		len(bc.Blocks[0].Metadata) {
		return fmt.Errorf(
			"incoming chain genesis metadata mismatch",
		)
	}

	for key, localValue := range bc.Blocks[0].Metadata {
		incomingValue, exists :=
			blocks[0].Metadata[key]

		if !exists ||
			incomingValue != localValue {
			return fmt.Errorf(
				"incoming chain genesis metadata mismatch for key %q",
				key,
			)
		}
	}

	// Eski/sentetik genesis hash'lerini yeni PoW bicimine zorlamiyoruz.
	// Yerel genesis kanonik PoW kullaniyorsa gelen genesis de kullanmalidir.
	if err := ValidateBlockPoWV1(
		bc.Blocks[0],
	); err == nil {
		if err := ValidateBlockPoWV1(
			blocks[0],
		); err != nil {
			return fmt.Errorf(
				"incoming chain genesis PoW validation failed: %w",
				err,
			)
		}
	}

	if !bytes.Equal(
		blocks[0].Hash,
		bc.Blocks[0].Hash,
	) {
		return fmt.Errorf(
			"incoming chain genesis hash mismatch",
		)
	}

	incomingMinted := 0

	for i, blk := range blocks {
		if blk == nil {
			return fmt.Errorf(
				"incoming chain nil block at position %d",
				i,
			)
		}

		if blk.Index != i {
			return fmt.Errorf(
				"incoming chain bad block index at position %d: got %d",
				i,
				blk.Index,
			)
		}

		if i > 0 {
			if err := ValidateBlockBasic(
				blk,
			); err != nil {
				return fmt.Errorf(
					"incoming chain block[%d] basic validation failed: %w",
					i,
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
					"incoming chain block[%d] protocol validation failed: %w",
					i,
					err,
				)
			}

			if blk.Difficulty < 1 ||
				blk.Difficulty > maxBits {
				return fmt.Errorf(
					"incoming chain block[%d] invalid difficulty: %d",
					i,
					blk.Difficulty,
				)
			}

			if !bytes.Equal(
				blk.PrevHash,
				blocks[i-1].Hash,
			) {
				return fmt.Errorf(
					"incoming chain prevhash mismatch at block[%d]",
					i,
				)
			}

			if blk.Timestamp <
				blocks[i-1].Timestamp {
				return fmt.Errorf(
					"incoming chain block[%d] timestamp goes backwards: got %d previous %d",
					i,
					blk.Timestamp,
					blocks[i-1].Timestamp,
				)
			}

		}

		if i == 0 {
			genesisMinted, err :=
				ValidateBlockSubsidyAgainstMintedV1(
					blk,
					incomingMinted,
					bc.TotalSupply,
				)
			if err != nil {
				return fmt.Errorf(
					"incoming chain block[%d] genesis allocation validation failed: %w",
					i,
					err,
				)
			}

			incomingMinted = genesisMinted
			continue
		}

		scratch := &Blockchain{
			Blocks: append(
				[]*Block(nil),
				blocks[:i]...,
			),
			UTXO:              map[string][]TransactionOutput{},
			TotalSupply:       bc.TotalSupply,
			coinbaseMaturity:  bc.coinbaseMaturity,
			activationPolicy:  bc.activationPolicy,
			nftAwardTrust:     bc.nftAwardTrust,
			nftMonthlyProgram: bc.nftMonthlyProgram,
		}

		scratch.UpdateUTXOSet()

		if err :=
			scratch.ValidateExpectedDifficultyV1(
				blk,
			); err != nil {
			return fmt.Errorf(
				"incoming chain block[%d] expected difficulty validation failed: %w",
				i,
				err,
			)
		}
		canonicalResult, err :=
			scratch.ValidateCanonicalBlockEnvelopeV1(
				blk,
				incomingMinted,
			)
		if err != nil {
			return fmt.Errorf(
				"incoming chain block[%d] canonical consensus validation failed: %w",
				i,
				err,
			)
		}

		protocol := scratch.ProtocolAtHeightV1(
			blk.Index,
		)

		rewardSubsidyQC, err :=
			ProtocolSubsidyUnitsToWholeQCV1(
				protocol,
				canonicalResult.Reward.Subsidy,
			)
		if err != nil {
			return fmt.Errorf(
				"incoming chain block[%d] subsidy accounting conversion failed: %w",
				i,
				err,
			)
		}

		nextMinted := incomingMinted + rewardSubsidyQC

		if nextMinted < incomingMinted {
			return fmt.Errorf(
				"incoming chain minted amount overflow",
			)
		}

		incomingMinted = nextMinted

		if bc.TotalSupply > 0 &&
			incomingMinted > bc.TotalSupply {
			return fmt.Errorf(
				"incoming chain exceeds total supply: minted=%d supply=%d",
				incomingMinted,
				bc.TotalSupply,
			)
		}

		if err :=
			scratch.ValidateFounderVestingBlockV1(
				blk,
			); err != nil {
			return fmt.Errorf(
				"incoming chain block[%d] founder vesting validation failed: %w",
				i,
				err,
			)
		}

		if err :=
			scratch.validateReplacementNormalTxSpendsV1(
				blk,
			); err != nil {
			return fmt.Errorf(
				"incoming chain block[%d] spend validation failed: %w",
				i,
				err,
			)
		}
	}

	return nil
}

func (bc *Blockchain) validateReplacementNormalTxSpendsV1(
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

	if _, err :=
		bc.ValidateTransactionsUTXOAtHeightV1(
			blk.Transactions,
			blk.Index,
		); err != nil {
		return fmt.Errorf(
			"replacement block canonical UTXO validation failed: %w",
			err,
		)
	}

	return nil
}

func (bc *Blockchain) ReplaceChain(
	blocks []*Block,
) error {
	return bc.ReplaceChainFromPeerStrictV1(
		blocks,
	)
}
