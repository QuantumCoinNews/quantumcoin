package miner

import (
	"fmt"

	"quantumcoin/blockchain"
	"quantumcoin/config"
)

const QCMinerRuntimeDifficultyVersionV1 = "QC_MINER_RUNTIME_DIFFICULTY_V1"

// resolveRuntimeDifficultyV1 applies the canonical network profile,
// dynamic chain difficulty, and the temporary local-release override
// through one shared policy.
//
// requestedBits <= 0 means:
// calculate the current dynamic value from the blockchain.
//
// requestedBits > 0 means:
// validate and use the explicitly supplied dynamic candidate.
func resolveRuntimeDifficultyV1(
	cfg *config.Config,
	bc *blockchain.Blockchain,
	requestedBits int,
) (int, error) {
	if cfg == nil {
		return 0, fmt.Errorf(
			"%s: config cannot be nil",
			QCMinerRuntimeDifficultyVersionV1,
		)
	}

	if err := config.ValidateRuntimeNetworkProfileV1(
		cfg,
	); err != nil {
		return 0, fmt.Errorf(
			"%s: runtime profile: %w",
			QCMinerRuntimeDifficultyVersionV1,
			err,
		)
	}

	dynamicBits := requestedBits

	if dynamicBits <= 0 {
		dynamicBits = cfg.DefaultDifficultyBits

		if bc != nil {
			dynamicBits = bc.NextDifficultySnapshot(
				cfg.DefaultDifficultyBits,
			)
		}
	}

	info, err := config.ResolveEffectiveDifficultyBitsV1(
		cfg,
		dynamicBits,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"%s: effective difficulty: %w",
			QCMinerRuntimeDifficultyVersionV1,
			err,
		)
	}

	return info.EffectiveBits, nil
}
