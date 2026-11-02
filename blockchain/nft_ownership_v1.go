package blockchain

import (
	"bytes"
	"fmt"
)

const NFTOwnershipVersionV1 = "QC_NFT_OWNERSHIP_V1"

// NFTOwnershipV1 describes an approved mint on the current canonical chain.
// Record.RecipientPubKeyHash is the owner. V1 has no transfer operation;
// changing a recipient requires another approved canonical history, not a
// local owner-map update. Token identity uses the existing NFTTokenIDV1 rule.
type NFTOwnershipV1 struct {
	TokenID       [32]byte
	Record        NFTMintRecordV1
	MintHeight    int
	MintBlockHash []byte
}

// NFTOwnershipSnapshotV1 binds all results to one canonical tip. Every byte
// slice is detached from live state and entries retain canonical block order.
type NFTOwnershipSnapshotV1 struct {
	Height  int
	TipHash []byte
	NFTs    []NFTOwnershipV1
}

// NFTOwnershipSnapshotSynchronizedV1 derives ownership from accepted block
// records under the canonical read lock. There is no separately persisted
// owner map, stale cache or abandoned-branch union. The already persisted
// records remain the source after trusted load, reload, reorg and rollback.
// Existing local NFT authorization is rechecked before returning any entry;
// a raw decoded chain cannot establish ownership from self-declared proofs.
// Block economics and admission remain the responsibility of the existing
// canonical write/load gates, not this read-only query.
func (bc *Blockchain) NFTOwnershipSnapshotSynchronizedV1() (NFTOwnershipSnapshotV1, error) {
	if bc == nil {
		return NFTOwnershipSnapshotV1{}, ErrNilBlockchain
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	if len(bc.Blocks) == 0 {
		return NFTOwnershipSnapshotV1{}, ErrEmptyChain
	}
	result := NFTOwnershipSnapshotV1{NFTs: []NFTOwnershipV1{}}
	seen := make(map[[32]byte]struct{})
	for index, block := range bc.Blocks {
		if block == nil || block.Index != index {
			return NFTOwnershipSnapshotV1{}, fmt.Errorf("%s: invalid canonical block at index %d", NFTOwnershipVersionV1, index)
		}
		if index > 0 && !bytes.Equal(block.PrevHash, bc.Blocks[index-1].Hash) {
			return NFTOwnershipSnapshotV1{}, fmt.Errorf("%s: disconnected block at index %d", NFTOwnershipVersionV1, index)
		}
		if len(block.NFTMintRecordsV1) == 0 && len(block.NFTAwardProofsV1) == 0 {
			continue
		}
		if err := bc.validateNFTMintWithLocalContextV1(bc.Blocks[:index], block); err != nil {
			return NFTOwnershipSnapshotV1{}, fmt.Errorf("%s: block[%d]: %w", NFTOwnershipVersionV1, index, err)
		}
		for _, record := range block.NFTMintRecordsV1 {
			id, err := NFTTokenIDV1(record)
			if err != nil {
				return NFTOwnershipSnapshotV1{}, fmt.Errorf("%s: token identity: %w", NFTOwnershipVersionV1, err)
			}
			if _, exists := seen[id]; exists {
				return NFTOwnershipSnapshotV1{}, fmt.Errorf("%s: duplicate canonical token", NFTOwnershipVersionV1)
			}
			seen[id] = struct{}{}
			record.RecipientPubKeyHash = cloneBytesV1(record.RecipientPubKeyHash)
			result.NFTs = append(result.NFTs, NFTOwnershipV1{
				TokenID: id, Record: record, MintHeight: block.Index,
				MintBlockHash: cloneBytesV1(block.Hash),
			})
		}
	}
	tip := bc.Blocks[len(bc.Blocks)-1]
	result.Height = tip.Index
	result.TipHash = cloneBytesV1(tip.Hash)
	return result, nil
}

// NFTByTokenIDSynchronizedV1 returns an independent canonical ownership entry.
// A missing token returns found=false, error=nil. Invalid history returns an
// error and never a partial ownership result.
func (bc *Blockchain) NFTByTokenIDSynchronizedV1(id [32]byte) (NFTOwnershipV1, bool, error) {
	snapshot, err := bc.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil {
		return NFTOwnershipV1{}, false, err
	}
	for _, entry := range snapshot.NFTs {
		if entry.TokenID == id {
			return entry, true, nil
		}
	}
	return NFTOwnershipV1{}, false, nil
}

// NFTsByOwnerSynchronizedV1 uses the canonical 20-byte recipient public-key
// hash, not a display address, miner name, Telegram ID or detached NFT database.
func (bc *Blockchain) NFTsByOwnerSynchronizedV1(ownerPubKeyHash []byte) ([]NFTOwnershipV1, error) {
	if bc == nil {
		return nil, ErrNilBlockchain
	}
	if len(ownerPubKeyHash) != 20 {
		return nil, fmt.Errorf("%s: owner public-key hash must be 20 bytes", NFTOwnershipVersionV1)
	}
	owner := cloneBytesV1(ownerPubKeyHash)
	snapshot, err := bc.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil {
		return nil, err
	}
	result := []NFTOwnershipV1{}
	for _, entry := range snapshot.NFTs {
		if bytes.Equal(entry.Record.RecipientPubKeyHash, owner) {
			result = append(result, entry)
		}
	}
	return result, nil
}
