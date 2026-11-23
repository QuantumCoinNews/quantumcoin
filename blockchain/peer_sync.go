package blockchain

import (
	"bytes"
	"fmt"
)

// PeerSyncRange copies only a bounded range from one coherent chain view.
// A continuation pins both the advertised tip and its previous block hash.
func (bc *Blockchain) PeerSyncRange(start, count, tipHeight int, tipHash, previous []byte) ([]*Block, int, []byte, error) {
	if bc == nil {
		return nil, 0, nil, ErrNilBlockchain
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	if start < 0 || count < 1 || count > 8 || len(bc.Blocks) == 0 {
		return nil, 0, nil, fmt.Errorf("invalid sync range")
	}
	if tipHeight == -1 {
		if start != 0 {
			return nil, 0, nil, fmt.Errorf("unanchored sync continuation")
		}
		tipHeight = len(bc.Blocks) - 1
		tipHash = bc.Blocks[tipHeight].Hash
	}
	if tipHeight < 0 || tipHeight >= len(bc.Blocks) || start > tipHeight ||
		!bytes.Equal(bc.Blocks[tipHeight].Hash, tipHash) {
		return nil, 0, nil, fmt.Errorf("sync tip no longer available")
	}
	if start > 0 && !bytes.Equal(bc.Blocks[start-1].Hash, previous) {
		return nil, 0, nil, fmt.Errorf("sync anchor mismatch")
	}
	end := min(start+count, tipHeight+1)
	return cloneBlocksV1(bc.Blocks[start:end]), tipHeight, cloneBytesV1(tipHash), nil
}

// NewPeerSyncCandidate validates the offered genesis against local state and
// seeds a detached replay engine with exclusively local consensus authorities.
func (bc *Blockchain) NewPeerSyncCandidate(genesis *Block) (*Blockchain, error) {
	if bc == nil {
		return nil, ErrNilBlockchain
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	if err := bc.validateCanonicalChainWithLocalContextV1([]*Block{genesis}); err != nil {
		return nil, err
	}
	result := &Blockchain{
		Blocks: cloneBlocksV1(bc.Blocks[:1]), TotalSupply: bc.TotalSupply,
		coinbaseMaturity: bc.coinbaseMaturity, activationPolicy: bc.activationPolicy,
		nftAwardTrust: bc.nftAwardTrust, nftMonthlyProgram: bc.nftMonthlyProgram,
		networkIdentity:         cloneNetworkIdentityV1(bc.networkIdentity),
		StoredNetworkIdentityV1: cloneNetworkIdentityV1(bc.StoredNetworkIdentityV1),
	}
	if result.nftAwardTrust.bound {
		result.nftAwardTrust.trust = cloneNFTAwardTrustV1(result.nftAwardTrust.trust)
	}
	result.UpdateUTXOSet()
	result.rebuildTransactionLocationIndexesLocked()
	return result, nil
}
