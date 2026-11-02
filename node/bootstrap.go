package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consensus"
	"quantumcoin/internal"
)

// Bootstrap loads configuration, opens or creates the blockchain, applies
// canonical consensus bindings and returns one shared node Runtime.
func Bootstrap(
	options BootstrapOptions,
) (*Runtime, error) {
	configPath := strings.TrimSpace(
		options.ConfigPath,
	)
	if configPath == "" {
		configPath = DefaultBootstrapOptions().
			ConfigPath
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: load config %q: %w",
			BootstrapVersionV1,
			configPath,
			err,
		)
	}

	return BootstrapWithConfig(
		cfg,
		options.StrictChain,
	)
}

// BootstrapWithConfig constructs a runtime from an already loaded config.
// It is also the testable canonical boundary used by Bootstrap.
func BootstrapWithConfig(
	cfg *config.Config,
	strictChain bool,
) (*Runtime, error) {
	release, snapshotDir, err := consensus.LoadCanonicalHardforkReleaseV1()
	if err != nil {
		return nil, err
	}
	return bootstrapWithHardforkReleaseV1(cfg, strictChain, release, snapshotDir)
}

// The release injection seam is private; production loads a detached manifest
// against the compiled trust root. Tests exercise the same startup with fixtures.
func bootstrapWithHardforkReleaseV1(
	cfg *config.Config,
	strictChain bool,
	release *consensus.HardforkReleaseV1,
	snapshotDir string,
) (*Runtime, error) {
	return bootstrapWithNFTStartupV1(cfg, strictChain, release, snapshotDir, consensus.CanonicalNFTAwardTrustV1(), filepath.Join(filepath.Dir(snapshotDir), consensus.NFTMonthlyProgramManifestNameV1))
}

// Only private test fixtures inject a trust root/path. Production obtains
// independent NFT authorities from the compiled canonical source.
func bootstrapWithNFTStartupV1(
	cfg *config.Config,
	strictChain bool,
	release *consensus.HardforkReleaseV1,
	snapshotDir string,
	nftTrust *consensus.NFTAwardTrustV1,
	nftManifest string,
) (*Runtime, error) {
	if cfg == nil {
		return nil, fmt.Errorf(
			"%s: config cannot be nil",
			BootstrapVersionV1,
		)
	}

	activation, err := consensus.LoadHardforkActivationV1(release, snapshotDir)
	if err != nil {
		return nil, fmt.Errorf("%s: hardfork activation: %w", BootstrapVersionV1, err)
	}
	if err := validateRuntimePolicyGateWithActivationV1(
		cfg,
		DefaultRuntimePolicySetV1(), activation,
	); err != nil {
		return nil, fmt.Errorf(
			"%s: runtime policy gate: %w",
			BootstrapVersionV1,
			err,
		)
	}

	if err := config.ValidateRuntimeNetworkProfileV1(
		cfg,
	); err != nil {
		return nil, fmt.Errorf(
			"%s: runtime network policy: %w",
			BootstrapVersionV1,
			err,
		)
	}

	if err :=
		config.ValidateTestnetFaucetPremineSafetyV1(); err != nil {
		return nil, fmt.Errorf(
			"%s: testnet faucet/premine guard: %w",
			BootstrapVersionV1,
			err,
		)
	}

	if err := config.ValidateChainFilePathForModeV1(
		cfg.ChainFile,
	); err != nil {
		return nil, fmt.Errorf(
			"%s: chain file mode guard: %w",
			BootstrapVersionV1,
			err,
		)
	}

	internal.SetBonusFile(
		cfg.BonusFile,
	)

	chain, nftProgram, err := loadBlockchainWithNFTStartupV1(cfg, activation, nftTrust, nftManifest)
	if err != nil {
		return nil, err
	}

	if strictChain {
		if err := chain.ValidateBlockchainSynchronized(); err != nil {
			return nil, fmt.Errorf(
				"%s: strict chain validation: %w",
				BootstrapVersionV1,
				err,
			)
		}
	}

	runtime := NewRuntime(chain, cfg)
	runtime.nftGameProgram = nftProgram
	return runtime, nil
}

func loadOrCreateBlockchainV1(
	cfg *config.Config,
) (*blockchain.Blockchain, error) {
	if cfg == nil {
		return nil, fmt.Errorf(
			"%s: config cannot be nil",
			BootstrapVersionV1,
		)
	}

	chainFile := strings.TrimSpace(
		cfg.ChainFile,
	)
	if chainFile == "" {
		return nil, fmt.Errorf(
			"%s: chain file cannot be empty",
			BootstrapVersionV1,
		)
	}

	_, statErr := os.Stat(chainFile)

	switch {
	case statErr == nil:
		chain, err :=
			blockchain.LoadBlockchainFromFile(
				chainFile,
			)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: load blockchain %q: %w",
				BootstrapVersionV1,
				chainFile,
				err,
			)
		}

		return chain, nil

	case errors.Is(statErr, os.ErrNotExist):
		return blockchain.NewBlockchain(
			cfg.InitialReward,
			cfg.TotalSupply,
		), nil

	default:
		return nil, fmt.Errorf(
			"%s: inspect chain file %q: %w",
			BootstrapVersionV1,
			chainFile,
			statErr,
		)
	}
}
