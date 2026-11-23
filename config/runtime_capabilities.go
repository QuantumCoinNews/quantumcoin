package config

import "quantumcoin/consensus"

// RuntimeCapabilitiesV1 reports implemented rules without consulting retired
// mutable globals. No staking yield or global mutable difficulty is advertised.
func RuntimeCapabilitiesV1() map[string]any {
	return map[string]any{
		"stakingEnabled": false, "stakingRewardRate": 0,
		"miningAvailable": true, "targetBlockSeconds": int(consensus.LegacyTargetBlockTime.Seconds()),
		"difficultySource":                           "validated-chain-and-network-profile",
		"consensusBlockSizeConfiguredByLegacyGlobal": false,
		"subsidySchedule":                            "WHOLE_QC_50_25_12_6_3_1",
		"fractionalSubsidyEnabled":                   false,
		"atomsPerQC":                                 consensus.CanonicalAtomsPerQCV1,
		"futureStakingRequiresConsensusActivation":   true,
		"legacyGlobalsAffectRuntime":                 false,
	}
}
