package blockchain

import (
	"fmt"

	qconsensus "quantumcoin/consensus"
)

func canonicalPoWHeaderV4(block *Block) (qconsensus.PoWHeaderV4, error) {
	if block == nil {
		return qconsensus.PoWHeaderV4{}, fmt.Errorf("canonical PoW header V4: block cannot be nil")
	}

	nftRecordDigest, err := NFTBlockCommitmentDigestV1(block)
	if err != nil {
		return qconsensus.PoWHeaderV4{}, fmt.Errorf("canonical PoW header V4: NFT records: %w", err)
	}

	return qconsensus.PoWHeaderV4{
		PreviousHash:      append([]byte(nil), block.PrevHash...),
		TransactionDigest: append([]byte(nil), block.HashTransactions()...),
		NFTRecordDigest:   append([]byte(nil), nftRecordDigest[:]...),
		Timestamp:         block.Timestamp,
		Difficulty:        block.Difficulty,
	}, nil
}
