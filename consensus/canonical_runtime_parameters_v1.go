package consensus

// These are existing protocol values, not a new activation or economic schedule.
// Config and presentation adapters must reference these constants. Network-specific
// port/difficulty overrides remain subject to the existing profile validators.
const (
	CanonicalGenesisUnixV1            int64  = 1_725_158_400
	CanonicalBlockTimeSecondsV1              = 30
	CanonicalHalvingIntervalBlocksV1  uint64 = 2_102_400
	CanonicalHalvingIntervalSecondsV1 int64  = int64(CanonicalHalvingIntervalBlocksV1) * CanonicalBlockTimeSecondsV1
	CanonicalMiningPeriodSecondsV1    int64  = 10 * 365 * 24 * 60 * 60 // Legacy config horizon, not emission cutoff.
	CanonicalDefaultDifficultyBitsV1         = 16
	CanonicalMinimumSubsidyQCV1       int64  = 1
	CanonicalMinerRewardPercentV1            = 100
	CanonicalStakeRewardPercentV1            = 0
	CanonicalDevRewardPercentV1              = 0
	CanonicalBurnRewardPercentV1             = 0
	CanonicalCommunityRewardPercentV1        = 0
)
