package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	qconsensus "quantumcoin/consensus"
)

// NewBlockAtProtocolV1 creates a block using the PoW rules selected for its
// consensus protocol. NFT mint records are only allowed under ProtocolNFTV4.
func NewBlockAtProtocolV1(
	index int,
	txs []*Transaction,
	prevHash []byte,
	miner string,
	difficulty int,
	nftRecords []NFTMintRecordV1,
	protocol qconsensus.ProtocolVersion,
) (*Block, error) {
	return newBlockAtProtocolAndTimeV1(index, txs, prevHash, miner, difficulty, nftRecords, protocol, time.Now().Unix())
}

func newBlockAtProtocolAndTimeV1(index int, txs []*Transaction, prevHash []byte, miner string, difficulty int, nftRecords []NFTMintRecordV1, protocol qconsensus.ProtocolVersion, timestamp int64) (*Block, error) {
	return newBlockAtProtocolAndTimeContextV1(context.Background(), index, txs, prevHash, miner, difficulty, nftRecords, protocol, timestamp)
}

func newBlockAtProtocolAndTimeContextV1(ctx context.Context, index int, txs []*Transaction, prevHash []byte, miner string, difficulty int, nftRecords []NFTMintRecordV1, protocol qconsensus.ProtocolVersion, timestamp int64) (*Block, error) {
	switch protocol {
	case qconsensus.ProtocolLegacyV1,
		qconsensus.ProtocolContinuityV2,
		qconsensus.ProtocolAtomV3:
		if len(nftRecords) != 0 {
			return nil, fmt.Errorf(
				"protocol %q cannot create uncommitted NFT records",
				protocol,
			)
		}
		return newBlockAtTimeContextV1(ctx, index, txs, prevHash, miner, difficulty, timestamp)

	case qconsensus.ProtocolNFTV4:
		return newNFTV4BlockContextV1(ctx, index, txs, prevHash, miner, difficulty, nftRecords, timestamp)

	default:
		return nil, fmt.Errorf("unsupported block protocol %q", protocol)
	}
}

func newNFTV4BlockContextV1(ctx context.Context, index int, txs []*Transaction, prevHash []byte, miner string, difficulty int, nftRecords []NFTMintRecordV1, timestamp int64) (*Block, error) {
	if _, err := NFTMintRecordsDigestV1(nftRecords); err != nil {
		return nil, fmt.Errorf("create NFT V4 block: %w", err)
	}

	recordsCopy := make([]NFTMintRecordV1, len(nftRecords))
	for i, record := range nftRecords {
		recordsCopy[i] = record
		recordsCopy[i].RecipientPubKeyHash = append(
			[]byte(nil),
			record.RecipientPubKeyHash...,
		)
	}

	block := &Block{
		Index:            index,
		Timestamp:        timestamp,
		Transactions:     txs,
		PrevHash:         append([]byte(nil), prevHash...),
		Miner:            miner,
		Difficulty:       difficulty,
		Metadata:         map[string]string{},
		NFTMintRecordsV1: recordsCopy,
	}

	header, err := canonicalPoWHeaderV4(block)
	if err != nil {
		return nil, fmt.Errorf("create NFT V4 block header: %w", err)
	}
	target := qconsensus.TargetFromDifficultyV1(difficulty)
	if target == nil {
		return nil, fmt.Errorf("create NFT V4 block: PoW target is unavailable")
	}

	for nonce := uint64(0); ; nonce++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		hash, err := qconsensus.ComputePoWHashV4(header, nonce)
		if err != nil {
			return nil, fmt.Errorf("create NFT V4 block PoW: %w", err)
		}

		var hashValue big.Int
		hashValue.SetBytes(hash[:])
		if hashValue.Cmp(target) < 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			nonceInt, err := ConsensusNonceIntV1(nonce)
			if err != nil {
				return nil, fmt.Errorf("create NFT V4 block nonce: %w", err)
			}
			block.Nonce = nonceInt
			block.Hash = append([]byte(nil), hash[:]...)

			if err := ValidateBlockPoWV4(block); err != nil {
				return nil, fmt.Errorf("validate created NFT V4 block: %w", err)
			}
			return block, nil
		}

		if nonce == qconsensus.MaxConsensusNonceV1 {
			break
		}
	}

	return nil, fmt.Errorf("create NFT V4 block: consensus nonce range exhausted")
}
