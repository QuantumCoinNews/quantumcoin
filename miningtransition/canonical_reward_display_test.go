package miningtransition

import (
	"encoding/json"
	"os"
	"quantumcoin/consensus"
	"reflect"
	"strconv"
	"testing"
)

func TestDisplayedRewardsMatchCanonicalWholeQCConsensus(t *testing.T) {
	for _, period := range append(PrimaryRewardScheduleV1(), PostTenYearExampleScheduleV1()...) {
		height := uint64(period.Index-1)*consensus.LegacyHalvingIntervalBlocks + 1
		expected, err := consensus.LegacySubsidyAtHeight(height)
		if err != nil {
			t.Fatal(err)
		}
		if period.RewardQC != strconv.FormatInt(expected, 10) {
			t.Fatal("display changed reward contract", period, expected)
		}
	}
	if rewardForHalvingV1(1000) != "1" || rewardForHalvingV1(-1) != "" {
		t.Fatal("unbounded/negative schedule input")
	}
	raw, err := os.ReadFile("../configs/mining_node_transition.example.json")
	if err != nil {
		t.Fatal(err)
	}
	var sample struct {
		PrimaryRewardSchedule      []RewardPeriod `json:"primaryRewardSchedule"`
		PostTenYearExampleSchedule []RewardPeriod `json:"postTenYearExampleSchedule"`
	}
	if err := json.Unmarshal(raw, &sample); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sample.PrimaryRewardSchedule, PrimaryRewardScheduleV1()) || !reflect.DeepEqual(sample.PostTenYearExampleSchedule, PostTenYearExampleScheduleV1()) {
		t.Fatal("example policy drifted from canonical subsidy")
	}
}
func TestMiningScheduleCannotAdvertiseFractionalOrClockAuthority(t *testing.T) {
	for _, mutate := range []func(*MiningNodeTransitionPolicy){func(p *MiningNodeTransitionPolicy) { p.ScheduleBasis = "FRACTIONAL" }, func(p *MiningNodeTransitionPolicy) { p.YearsAreIllustrative = false }, func(p *MiningNodeTransitionPolicy) { p.MinimumSubsidyQC = 0 }, func(p *MiningNodeTransitionPolicy) { p.PrimaryRewardSchedule[2].RewardQC = "12.5" }} {
		p := DefaultPolicyV1()
		mutate(&p)
		if err := ValidatePolicyV1(p); err == nil {
			t.Fatal("wrong reward semantics accepted")
		}
	}
}
