package node

import (
	"fmt"
	"strings"

	"quantumcoin/blockchain"
	"quantumcoin/consensus"
)

// MineNFTMonthlyAwardAndPersistV1 is the runtime's controlled write boundary.
// Public signed evidence cannot install authorities, program dates or keys.
// The runtime must own a verified startup program and a writable chain file.
// It returns only after the shared canonical engine's persistence succeeds.
func (r *Runtime) MineNFTMonthlyAwardAndPersistV1(
	miner string,
	record blockchain.NFTMintRecordV1,
	proof blockchain.NFTAwardProofV1,
) (*blockchain.Block, error) {
	if r == nil {
		return nil, fmt.Errorf("persistent NFT runtime: unavailable")
	}
	r.miningMu.Lock()
	defer r.miningMu.Unlock()
	// Keep chain, mode, configuration and the startup program stable until
	// the writer returns. Do not call another runtime RLock method here.
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.APIReadOnly {
		return nil, fmt.Errorf("persistent NFT runtime: API is read-only")
	}
	if r.Blockchain == nil || r.Config == nil {
		return nil, fmt.Errorf("persistent NFT runtime: chain/config unavailable")
	}
	filename := strings.TrimSpace(r.Config.ChainFile)
	if filename == "" {
		return nil, fmt.Errorf("persistent NFT runtime: chain filename is empty")
	}
	program := r.nftGameProgram
	if !program.VerifiedV1() {
		return nil, fmt.Errorf("persistent NFT runtime: verified startup program required")
	}
	policy, trust := program.PolicySnapshotV1(), program.TrustSnapshotV1()
	activation := r.Blockchain.ActivationPolicySnapshotV1()
	if err := activation.Validate(); err != nil || !activation.Enabled ||
		activation.Plan.EffectiveTargetProtocol() != consensus.ProtocolNFTV4 ||
		activation.Plan.NetworkID != policy.NetworkID ||
		!strings.EqualFold(activation.Plan.LegacyGenesisHex, trust.LegacyGenesisHex) ||
		!r.Blockchain.NFTAwardTrustBoundSnapshotV1() {
		return nil, fmt.Errorf("persistent NFT runtime: local activation/trust differs from startup program")
	}
	serial, err := policy.SerialForAwardPeriodV1(record.AwardPeriod)
	if err != nil || record.NetworkID != policy.NetworkID ||
		record.CampaignID != policy.CampaignID || record.Serial != serial {
		return nil, fmt.Errorf("persistent NFT runtime: award differs from verified startup program")
	}
	return r.Blockchain.MineBlockWithNFTAwardAndPersistSynchronizedV1(miner, record, proof, filename)
}
