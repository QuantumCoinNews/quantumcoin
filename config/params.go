package config

import qconsensus "quantumcoin/consensus"

// Retired compatibility variables; changing them has no runtime/consensus effect.
// Deprecated: consult RuntimeCapabilitiesV1 and canonical consensus parameters.
var (
	BlockTimeSeconds        = qconsensus.CanonicalBlockTimeSecondsV1      // Retired placeholder; consensus owns the target interval.
	MaxBlockSizeBytes       = 1_000_000                                   // Retired placeholder; does not set a consensus block-size rule.
	EnableStaking           = false                                       // Retired: staking is unavailable.
	EnableMining            = true                                        // Retired placeholder; does not start or stop mining.
	CurrentMiningDifficulty = qconsensus.CanonicalDefaultDifficultyBitsV1 // Retired placeholder; validated chain/profile owns difficulty.
	CurrentStakeRewardRate  = 0.0                                         // Retired: no staking emission.
)
