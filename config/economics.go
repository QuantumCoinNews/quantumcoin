package config

import qconsensus "quantumcoin/consensus"

// These shares describe the implemented coinbase recipient, not founder allocation.
// All subsidy and validated fees go to the miner; founder vesting is separate.
const (
	ShareMiningPct    = qconsensus.CanonicalMinerRewardPercentV1
	ShareStakingPct   = qconsensus.CanonicalStakeRewardPercentV1
	ShareDevPct       = qconsensus.CanonicalDevRewardPercentV1
	ShareBurnPct      = qconsensus.CanonicalBurnRewardPercentV1
	ShareCommunityPct = qconsensus.CanonicalCommunityRewardPercentV1
)
