package consensus_test

import (
	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consensus"
	"testing"
	"time"
)

func TestRuntimeAndMainnetParameterAdaptersUseCanonicalValues(t *testing.T) {
	check := func(want int64, values ...int64) bool {
		for _, v := range values {
			if v != want {
				return true
			}
		}
		return false
	}
	c := config.Default()
	if check(consensus.CanonicalGenesisUnixV1, c.GenesisUnix, blockchain.CanonicalGenesisUnixV1, config.QCMainnetImmutableGenesisUnixV1) {
		t.Fatal("genesis drift")
	}
	if check(consensus.CanonicalBlockTimeSecondsV1, int64(c.TargetBlockTimeSecs), int64(config.QCMainnetImmutableTargetBlockTimeSecsV1), int64(consensus.LegacyTargetBlockTime/time.Second)) {
		t.Fatal("interval drift")
	}
	if check(consensus.CanonicalHalvingIntervalSecondsV1, c.HalvingIntervalSecs, config.QCMainnetImmutableHalvingIntervalSecsV1) {
		t.Fatal("halving drift")
	}
	if check(int64(consensus.CanonicalHalvingIntervalBlocksV1), int64(blockchain.CanonicalBlocksPerHalvingV1), int64(consensus.LegacyHalvingIntervalBlocks)) {
		t.Fatal("block schedule drift")
	}
	if check(consensus.CanonicalMiningPeriodSecondsV1, c.MiningPeriodSecs, config.QCMainnetImmutableMiningPeriodSecsV1) {
		t.Fatal("horizon drift")
	}
	if check(consensus.CanonicalDefaultDifficultyBitsV1, int64(c.DefaultDifficultyBits), int64(config.QCMainnetImmutableDefaultDifficultyBitsV1)) {
		t.Fatal("difficulty drift")
	}

	for i, want := range []int{50, 25, 12, 6, 3, 1, 1} {
		h := i*blockchain.CanonicalBlocksPerHalvingV1 + 1
		got, err := consensus.LegacySubsidyAtHeight(uint64(h))
		if err != nil || got != int64(want) || blockchain.RewardAtHeightV1(h) != want {
			t.Fatal("whole QC reward changed", h, got, err)
		}
	}
	if config.ShareMiningPct != consensus.CanonicalMinerRewardPercentV1 || config.ShareStakingPct+config.ShareDevPct+config.ShareBurnPct+config.ShareCommunityPct != 0 {
		t.Fatal("coinbase recipient drift")
	}
}
