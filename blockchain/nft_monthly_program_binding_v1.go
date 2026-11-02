package blockchain

import (
	"errors"
	"fmt"
	qconsensus "quantumcoin/consensus"
)

var ErrNFTMonthlyProgramUnboundV1 = errors.New("NFT monthly program: local program is not bound")

type nftMonthlyProgramBindingStateV1 struct {
	bound  bool
	policy qconsensus.NFTMonthlyProgramPolicyV1
}

// This binds validated policy shape from trusted local game configuration,
// never from a peer or an award. It does not authenticate a program release,
// select a user or enable block admission. No date or real keys are chosen.
func (bc *Blockchain) BindLocalNFTMonthlyProgramSynchronizedV1(policy qconsensus.NFTMonthlyProgramPolicyV1) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	if bc.nftMonthlyProgram.bound {
		return fmt.Errorf("NFT monthly program: policy already bound")
	}
	if !bc.nftAwardTrust.bound {
		return ErrNFTAwardTrustUnboundV1
	}
	if err := policy.Validate(); err != nil {
		return err
	}
	if policy.NetworkID != bc.nftAwardTrust.trust.NetworkID {
		return fmt.Errorf("NFT monthly program: differs from local authority network")
	}
	bc.nftMonthlyProgram = nftMonthlyProgramBindingStateV1{bound: true, policy: policy}
	return nil
}

// validateNFTMonthlyAwardBlockLockedV1 is a preflight check, not live admission.
// The caller holds the chain lock. Existing history and the candidate must be
// validated together; there is no RAM-only award counter to lose on restart.
func (bc *Blockchain) validateNFTMonthlyAwardBlockLockedV1(candidate *Block) error {
	if candidate == nil {
		return ErrNilBlock
	}
	if len(candidate.NFTMintRecordsV1) == 0 && len(candidate.NFTAwardProofsV1) == 0 {
		return nil
	}
	if !bc.nftMonthlyProgram.bound {
		return ErrNFTMonthlyProgramUnboundV1
	}
	if !bc.nftAwardTrust.bound {
		return ErrNFTAwardTrustUnboundV1
	}
	policy := bc.nftMonthlyProgram.policy
	count := 0
	check := func(block *Block) error {
		if block == nil {
			return ErrNilBlock
		}
		if len(block.NFTMintRecordsV1) == 0 && len(block.NFTAwardProofsV1) == 0 {
			return nil
		}
		if block.Index <= 0 {
			return fmt.Errorf("NFT monthly program: genesis awards are forbidden")
		}
		if len(block.NFTMintRecordsV1) == 0 || len(block.NFTMintRecordsV1) != len(block.NFTAwardProofsV1) {
			return fmt.Errorf("NFT monthly program: every record needs an approval proof")
		}
		if err := ValidateBlockPoWV4(block); err != nil {
			return fmt.Errorf("NFT monthly program: block commitment: %w", err)
		}
		for i, record := range block.NFTMintRecordsV1 {
			if record.NetworkID != policy.NetworkID || record.CampaignID != policy.CampaignID {
				return fmt.Errorf("NFT monthly program: record differs from local network/campaign")
			}
			expected, err := policy.SerialForAwardPeriodV1(record.AwardPeriod)
			if err != nil {
				return err
			}
			if record.Serial != expected {
				return fmt.Errorf("NFT monthly program: serial %d differs from month serial %d", record.Serial, expected)
			}
			proof := block.NFTAwardProofsV1[i]
			if err := bc.validateNFTAwardWithLocalTrustLockedV1(record, proof.Decision, proof.Approvals); err != nil {
				return err
			}
			count++
			if count > qconsensus.MonthlyRareNFTAwardCountV1 {
				return fmt.Errorf("NFT monthly program: total exceeds twelve awards")
			}
		}
		return nil
	}
	for _, existing := range bc.Blocks {
		if err := check(existing); err != nil {
			return err
		}
	}
	if err := check(candidate); err != nil {
		return err
	}
	return validateNFTMintUniquenessV1(bc.Blocks, candidate)
}

// ValidateNFTMonthlyAwardBlockSynchronizedV1 checks signed, PoW-bound monthly
// evidence against local program and chain history. It does not prove active
// Telegram participation/randomness, append a block, mint or record ownership.
// Activation and full transaction/consensus validation are separate gates.
func (bc *Blockchain) ValidateNFTMonthlyAwardBlockSynchronizedV1(candidate *Block) error {
	if bc == nil {
		return ErrNilBlockchain
	}
	bc.stateMu.RLock()
	defer bc.stateMu.RUnlock()
	return bc.validateNFTMonthlyAwardBlockLockedV1(candidate)
}
