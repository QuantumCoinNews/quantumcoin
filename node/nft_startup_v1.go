package node

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consensus"
)

type NFTGameProgramSnapshotV1 struct {
	Verified        bool                                `json:"verified"`
	GameActivatedAt int64                               `json:"gameActivatedAt"`
	Policy          consensus.NFTMonthlyProgramPolicyV1 `json:"policy"`
}

// Read-only source for the game bridge. It exposes no authority keys and only
// reports a game date established by this runtime's verified startup program.
func (r *Runtime) NFTGameProgramSnapshotV1() NFTGameProgramSnapshotV1 {
	if r == nil {
		return NFTGameProgramSnapshotV1{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p := r.nftGameProgram
	if !p.VerifiedV1() {
		return NFTGameProgramSnapshotV1{}
	}
	return NFTGameProgramSnapshotV1{Verified: true, GameActivatedAt: p.GameActivatedAtV1(), Policy: p.PolicySnapshotV1()}
}

// Startup binds private local policies BEFORE decoding an NFT-bearing file.
// No runtime is published, file written or service started on any failure.
func loadBlockchainWithNFTStartupV1(cfg *config.Config, activation consensus.ActivationPolicyV1, trust *consensus.NFTAwardTrustV1, manifest string) (*blockchain.Blockchain, consensus.VerifiedNFTMonthlyProgramV1, error) {
	var zero consensus.VerifiedNFTMonthlyProgramV1
	if err := activation.Validate(); err != nil {
		return nil, zero, err
	}
	if !activation.Enabled || activation.Plan.EffectiveTargetProtocol() != consensus.ProtocolNFTV4 {
		chain, err := loadOrCreateBlockchainV1(cfg)
		if err != nil {
			return nil, zero, err
		}
		if err := chain.BindActivationPolicySynchronizedV1(activation); err != nil {
			return nil, zero, err
		}
		chain.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
		return chain, zero, nil
	}
	if trust == nil {
		return nil, zero, fmt.Errorf("NFT startup: local NFT authorities/program are not finalized")
	}
	program, err := consensus.LoadNFTMonthlyProgramManifestV1(manifest, *trust)
	if err != nil {
		return nil, zero, fmt.Errorf("NFT startup: %w", err)
	}
	local := program.TrustSnapshotV1()
	if local.NetworkID != activation.Plan.NetworkID || !strings.EqualFold(local.LegacyGenesisHex, activation.Plan.LegacyGenesisHex) {
		return nil, zero, fmt.Errorf("NFT startup: program differs from verified activation network/genesis")
	}
	context := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	if err := context.BindActivationPolicySynchronizedV1(activation); err != nil {
		return nil, zero, err
	}
	if err := context.BindLocalNFTAwardTrustSynchronizedV1(local.NetworkID, local); err != nil {
		return nil, zero, err
	}
	if err := context.BindLocalNFTMonthlyProgramSynchronizedV1(program.PolicySnapshotV1()); err != nil {
		return nil, zero, err
	}
	context.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
	chainFile := strings.TrimSpace(cfg.ChainFile)
	if chainFile == "" {
		return nil, zero, fmt.Errorf("NFT startup: chain file cannot be empty")
	}
	_, statErr := os.Stat(chainFile)
	if errors.Is(statErr, os.ErrNotExist) {
		return context, program, nil
	}
	if statErr != nil {
		return nil, zero, fmt.Errorf("NFT startup: inspect chain: %w", statErr)
	}
	chain, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(chainFile, context)
	if err != nil {
		return nil, zero, fmt.Errorf("NFT startup: load trusted chain: %w", err)
	}
	return chain, program, nil
}
