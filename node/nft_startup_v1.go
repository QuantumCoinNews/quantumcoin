package node

import (
	"fmt"
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

// NFTGameProgramSnapshotV1 Read-only source for the game bridge. It exposes no authority keys and only
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
		// Establish trusted local genesis/policies before decoding any persisted
		// blocks. File-supplied genesis, supply and UTXO are never startup authority.
		context, err := newRuntimeChainContextV1(cfg)
		if err != nil {
			return nil, zero, err
		}
		if err := context.BindActivationPolicySynchronizedV1(activation); err != nil {
			return nil, zero, err
		}
		context.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
		chain, err := loadOrCreateBlockchainWithContextV1(cfg, context)
		return chain, zero, err
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
	context, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		return nil, zero, err
	}
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
	chain, err := loadOrCreateBlockchainWithContextV1(cfg, context)
	if err != nil {
		return nil, zero, fmt.Errorf("NFT startup: load trusted chain: %w", err)
	}
	return chain, program, nil
}

func newRuntimeChainContextV1(cfg *config.Config) (*blockchain.Blockchain, error) {
	profile, err := config.ResolveNetworkProfileV1(cfg.ChainFile)
	if err != nil {
		return nil, err
	}
	if err := config.ValidateGenesisProfileV1(cfg.GenesisProfile, profile.Profile); err != nil {
		return nil, err
	}
	if profile.Profile == config.QCNetworkProfileTestnetPublicV1 && cfg.GenesisProfile != config.NetworkGenesisProfileV4 {
		return nil, fmt.Errorf("public testnet requires network-v4 genesis-bound transaction signatures; retain legacy chains in a separate local profile")
	}
	if profile.Profile == config.QCNetworkProfileMainnetV1 && cfg.GenesisProfile != config.NetworkGenesisProfileV4 {
		return nil, fmt.Errorf("mainnet starts fresh with network-v4; historical chains cannot be used as the new mainnet")
	}
	var chain *blockchain.Blockchain
	if cfg.GenesisProfile == config.NetworkGenesisProfileV4 && profile.IsTestnet {
		if err := config.ValidateFreshTestnetGenesisInputs(cfg); err != nil {
			return nil, err
		}
	}
	switch {
	case cfg.GenesisProfile == config.NetworkGenesisProfileV4:
		chain, err = blockchain.NewNetworkBlockchainV4(cfg, profile.Profile)
	case cfg.GenesisProfile == config.NetworkGenesisProfileV3:
		chain, err = blockchain.NewNetworkBlockchainV3(cfg, profile.Profile)
	case cfg.GenesisProfile == config.NetworkGenesisProfileV2:
		chain, err = blockchain.NewNetworkBlockchainV2(cfg, profile.Profile)
	case profile.ExpectedChainFile == config.QCIsolatedLocalChainFileV1:
		chain = blockchain.NewIsolatedLocalBlockchainV1(cfg.InitialReward, cfg.TotalSupply)
	default:
		chain = blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	}
	if err != nil {
		return nil, err
	}
	if err := chain.BindRuntimeNetworkIdentityV1(string(profile.Profile)); err != nil {
		return nil, err
	}
	return chain, nil
}
