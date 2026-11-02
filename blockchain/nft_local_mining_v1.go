package blockchain

import (
	"fmt"
	"time"

	qconsensus "quantumcoin/consensus"
)

// MineBlockWithNFTAwardSynchronizedV1 mines one locally approved monthly award
// through the existing canonical reward/fee/mempool engine. The submitted
// proof is public evidence; it cannot supply local activation or authorities.
// This entry does not select a Telegram user or enable the real game program.
func (bc *Blockchain) MineBlockWithNFTAwardSynchronizedV1(
	miner string,
	difficulty int,
	record NFTMintRecordV1,
	proof NFTAwardProofV1,
) (*Block, error) {
	if bc == nil {
		return nil, ErrChainNotInitialized
	}
	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()
	return bc.mineNFTAwardWithCommitLockedV1(miner, difficulty, record, proof, "")
}

// Shared approval/program preflight; its caller owns stateMu.
func (bc *Blockchain) mineNFTAwardWithCommitLockedV1(miner string, difficulty int, record NFTMintRecordV1, proof NFTAwardProofV1, chainFile string) (*Block, error) {
	if len(bc.Blocks) == 0 || bc.Blocks[len(bc.Blocks)-1] == nil {
		return nil, ErrChainNotInitialized
	}
	height := bc.Blocks[len(bc.Blocks)-1].Index + 1
	if bc.ProtocolAtHeightV1(height) != qconsensus.ProtocolNFTV4 {
		return nil, fmt.Errorf("local NFT mining: verified NFTV4 activation required")
	}
	if !bc.nftAwardTrust.bound {
		return nil, ErrNFTAwardTrustUnboundV1
	}
	if !bc.nftMonthlyProgram.bound {
		return nil, ErrNFTMonthlyProgramUnboundV1
	}
	if bc.activationPolicy.policy.Plan.NetworkID != bc.nftAwardTrust.trust.NetworkID ||
		bc.nftMonthlyProgram.policy.NetworkID != bc.nftAwardTrust.trust.NetworkID {
		return nil, fmt.Errorf("local NFT mining: activation, authority and program networks differ")
	}
	evidence := cloneBlockV1(&Block{NFTMintRecordsV1: []NFTMintRecordV1{record}, NFTAwardProofsV1: []NFTAwardProofV1{proof}})
	record, proof = evidence.NFTMintRecordsV1[0], evidence.NFTAwardProofsV1[0]
	// Cheap rejection before PoW. Full local-context, PoW and economic checks
	// still run at the existing strict commit gate under this same writer lock.
	if err := bc.validateNFTAwardWithLocalTrustLockedV1(record, proof.Decision, proof.Approvals); err != nil {
		return nil, err
	}
	policy := bc.nftMonthlyProgram.policy
	serial, err := policy.SerialForAwardPeriodV1(record.AwardPeriod)
	if err != nil {
		return nil, err
	}
	if record.NetworkID != policy.NetworkID || record.CampaignID != policy.CampaignID || record.Serial != serial {
		return nil, fmt.Errorf("local NFT mining: award differs from the local monthly program")
	}
	if record.AwardPeriod != time.Now().UTC().Format("2006-01") {
		return nil, fmt.Errorf("local NFT mining: award is outside the current UTC month")
	}
	if err := validateNFTMintUniquenessV1(bc.Blocks, evidence); err != nil {
		return nil, err
	}
	return bc.mineBlockWithNFTCommitLockedV1(miner, difficulty, evidence.NFTMintRecordsV1, evidence.NFTAwardProofsV1, chainFile)
}
