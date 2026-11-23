package config

import "testing"

func TestMiningRecipientContractIgnoresLegacyProposal(t *testing.T) {
	defaults := Default()
	if defaults.RewardPctMiner != 100 || defaults.RewardPctStake != 0 || defaults.RewardPctDev != 0 || defaults.RewardPctBurn != 0 || defaults.CommunityPercent() != 0 {
		t.Fatal("defaults imply unsupported block splits", defaults)
	}
	defaults.RewardPctMiner = 70
	defaults.RewardPctStake = 10
	defaults.RewardPctDev = 10
	defaults.RewardPctBurn = 5
	if defaults.CommunityPercent() != 0 {
		t.Fatal("legacy proposal claimed community payment")
	}
}
