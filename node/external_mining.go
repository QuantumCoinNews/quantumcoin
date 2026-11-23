package node

import (
	"crypto/sha256"
	"fmt"
	"quantumcoin/blockchain"
	"quantumcoin/config"
)

func (r *Runtime) PrepareExternalMining(miner string) (*blockchain.Block, []byte, error) {
	if r == nil {
		return nil, nil, ErrMiningBlockchainUnavailableV1
	}
	r.miningMu.Lock()
	defer r.miningMu.Unlock()
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.stopping {
		return nil, nil, ErrRuntimeShuttingDownV1
	}
	if r.APIReadOnly {
		return nil, nil, ErrRuntimeReadOnlyV1
	}
	if r.Blockchain == nil || r.Config == nil {
		return nil, nil, ErrMiningBlockchainUnavailableV1
	}
	dynamic := r.Blockchain.NextDifficultySnapshot(r.Config.DefaultDifficultyBits)
	policy, err := config.ResolveEffectiveDifficultyBitsV1(r.Config, dynamic)
	if err != nil {
		return nil, nil, err
	}
	return r.Blockchain.PrepareExternalMiningTemplate(miner, policy.EffectiveBits)
}

// CommitExternalMining verifies the supplied nonce and persists that very block.
// It never searches for a replacement nonce. A nil error means local disk commit.
func (r *Runtime) CommitExternalMining(template *blockchain.Block, nonce uint64) (result *blockchain.Block, err error) {
	defer func() {
		if result != nil && err == nil {
			r.recordAdvisoryBlock(result.Index)
			r.persistAdvisory()
		}
	}()
	if r == nil {
		return nil, ErrMiningBlockchainUnavailableV1
	}
	r.miningMu.Lock()
	defer r.miningMu.Unlock()
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.stopping {
		return nil, ErrRuntimeShuttingDownV1
	}
	if r.APIReadOnly {
		return nil, ErrRuntimeReadOnlyV1
	}
	if r.Blockchain == nil || r.Config == nil {
		return nil, ErrMiningBlockchainUnavailableV1
	}
	if template == nil {
		return nil, blockchain.ErrNilBlock
	}
	// Detach from the issued template; a failed disk write remains retryable.
	block := blockchain.DeserializeBlock(template.Serialize())
	n, err := blockchain.ConsensusNonceIntV1(nonce)
	if err != nil {
		return nil, err
	}
	header, err := r.Blockchain.SerializeMiningHeader(block, nonce)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(header)
	block.Nonce = n
	block.Hash = hash[:]
	if err := r.Blockchain.CommitValidatedBlockAndPersist(block, r.Config.ChainFile); err != nil {
		return nil, fmt.Errorf("external mining commit: %w", err)
	}
	return block, nil
}
