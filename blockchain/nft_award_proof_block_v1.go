package blockchain

import (
	"fmt"
	"math/big"
	"time"

	qconsensus "quantumcoin/consensus"
)

// NewNFTBlockWithAwardProofV1 constructs a detached V4 PoW block containing
// approval evidence before mining. It does not append to any chain or approve
// a signer set. All live NFT admission/mint gates remain closed until runtime
// authorization, program rules and ownership are integrated.
func NewNFTBlockWithAwardProofV1(
	index int,
	txs []*Transaction,
	prevHash []byte,
	miner string,
	difficulty int,
	records []NFTMintRecordV1,
	proofs []NFTAwardProofV1,
) (*Block, error) {
	if index <= 0 || len(records) == 0 || len(proofs) == 0 {
		return nil, fmt.Errorf("create approved NFT block: non-genesis height, records and proofs are required")
	}
	block := cloneBlockV1(&Block{
		Index: index, Timestamp: time.Now().Unix(), Transactions: txs,
		PrevHash: prevHash, Miner: miner, Difficulty: difficulty,
		Metadata: map[string]string{}, NFTMintRecordsV1: records, NFTAwardProofsV1: proofs,
	})
	header, err := canonicalPoWHeaderV4(block)
	if err != nil {
		return nil, fmt.Errorf("create approved NFT block: %w", err)
	}
	target := qconsensus.TargetFromDifficultyV1(difficulty)
	if target == nil {
		return nil, fmt.Errorf("create approved NFT block: PoW target is unavailable")
	}
	for nonce := uint64(0); ; nonce++ {
		hash, err := qconsensus.ComputePoWHashV4(header, nonce)
		if err != nil {
			return nil, err
		}
		var value big.Int
		value.SetBytes(hash[:])
		if value.Cmp(target) < 0 {
			n, err := ConsensusNonceIntV1(nonce)
			if err != nil {
				return nil, err
			}
			block.Nonce = n
			block.Hash = append([]byte(nil), hash[:]...)
			if err := ValidateBlockPoWV4(block); err != nil {
				return nil, err
			}
			return block, nil
		}
		if nonce == qconsensus.MaxConsensusNonceV1 {
			break
		}
	}
	return nil, fmt.Errorf("create approved NFT block: consensus nonce range exhausted")
}
