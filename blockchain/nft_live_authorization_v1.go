package blockchain

import (
	"fmt"
	"time"

	qconsensus "quantumcoin/consensus"
)

// validateNFTMintWithLocalContextV1 is the chain-state NFT admission gate.
// The caller owns the chain lock (or a detached validation scratch chain).
// All runtime policies are private local state, never supplied by a block.
// The stateless compatibility guard still fails closed without this context.
func (bc *Blockchain) validateNFTMintWithLocalContextV1(history []*Block, candidate *Block) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	if candidate == nil {
		return ErrNilBlock
	}
	if len(candidate.NFTMintRecordsV1) == 0 && len(candidate.NFTAwardProofsV1) == 0 {
		return nil
	}
	protocol := bc.ProtocolAtHeightV1(candidate.Index)
	if protocol != qconsensus.ProtocolNFTV4 {
		return validateNFTMintAuthorizationV1(candidate, protocol)
	}
	if candidate.Index <= 0 {
		return fmt.Errorf("NFT runtime admission: genesis awards are forbidden")
	}
	if !bc.activationPolicy.bound || !bc.activationPolicy.policy.Enabled {
		return fmt.Errorf("NFT runtime admission: verified local activation required")
	}
	if err := bc.activationPolicy.policy.Validate(); err != nil {
		return fmt.Errorf("NFT runtime admission: local activation: %w", err)
	}
	if !bc.nftAwardTrust.bound {
		return ErrNFTAwardTrustUnboundV1
	}
	if !bc.nftMonthlyProgram.bound {
		return ErrNFTMonthlyProgramUnboundV1
	}
	if bc.activationPolicy.policy.Plan.NetworkID != bc.nftAwardTrust.trust.NetworkID ||
		bc.nftMonthlyProgram.policy.NetworkID != bc.nftAwardTrust.trust.NetworkID {
		return fmt.Errorf("NFT runtime admission: activation, authority and program network differ")
	}
	checkMonth := func(block *Block) error {
		if block == nil {
			return ErrNilBlock
		}
		if len(block.NFTMintRecordsV1) == 0 && len(block.NFTAwardProofsV1) == 0 {
			return nil
		}
		if block.Index <= 0 || bc.ProtocolAtHeightV1(block.Index) != qconsensus.ProtocolNFTV4 {
			return fmt.Errorf("NFT runtime admission: award precedes verified NFT activation")
		}
		if block.Timestamp <= 0 {
			return fmt.Errorf("NFT runtime admission: invalid block timestamp")
		}
		month := time.Unix(block.Timestamp, 0).UTC().Format("2006-01")
		for _, record := range block.NFTMintRecordsV1 {
			if record.AwardPeriod != month {
				return fmt.Errorf("NFT runtime admission: award period %q differs from block UTC month %q", record.AwardPeriod, month)
			}
		}
		return nil
	}
	for _, block := range history {
		if err := checkMonth(block); err != nil {
			return err
		}
	}
	if err := checkMonth(candidate); err != nil {
		return err
	}
	scratch := &Blockchain{
		Blocks:            history,
		activationPolicy:  bc.activationPolicy,
		nftAwardTrust:     bc.nftAwardTrust,
		nftMonthlyProgram: bc.nftMonthlyProgram,
	}
	// Reuse signatures, program scope/serial, PoW commitment, history count
	// and one-award-per-network/month rules. No second fee/reward algorithm.
	return scratch.validateNFTMonthlyAwardBlockLockedV1(candidate)
}
