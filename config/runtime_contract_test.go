package config

import (
	"quantumcoin/consensus"
	"reflect"
	"testing"
)

func TestRewardRoleValidationRejectsOverflowAndInactiveSplits(t *testing.T) {
	for _, roles := range [][4]int{{101, 0, 0, 0}, {100, 1, 0, 0}, {-1, 0, 0, 0}, {70, 10, 10, 5}, {70, 10, 10, 10}, {99, 0, 0, 0}, {int(^uint(0) >> 1), 1, 0, 0}} {
		c := Default()
		c.RewardPctMiner = roles[0]
		c.RewardPctStake = roles[1]
		c.RewardPctDev = roles[2]
		c.RewardPctBurn = roles[3]
		if err := c.Validate(); err == nil {
			t.Fatal("invalid role config accepted", roles)
		}
	}
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestRetiredGlobalsCannotEnableRuntimeFeatures(t *testing.T) {
	before := RuntimeCapabilitiesV1()
	staking, rate, difficulty, seconds, size, mining := EnableStaking, CurrentStakeRewardRate, CurrentMiningDifficulty, BlockTimeSeconds, MaxBlockSizeBytes, EnableMining
	defer func() {
		EnableStaking = staking
		CurrentStakeRewardRate = rate
		CurrentMiningDifficulty = difficulty
		BlockTimeSeconds = seconds
		MaxBlockSizeBytes = size
		EnableMining = mining
	}()
	EnableStaking = true
	CurrentStakeRewardRate = 999
	CurrentMiningDifficulty = 1
	BlockTimeSeconds = 1
	MaxBlockSizeBytes = 1
	EnableMining = false
	if !reflect.DeepEqual(before, RuntimeCapabilitiesV1()) {
		t.Fatal("legacy globals changed runtime capability contract")
	}
	if QC != consensus.CanonicalAtomsPerQCV1 {
		t.Fatal("config scale differs from canonical atoms")
	}
}
