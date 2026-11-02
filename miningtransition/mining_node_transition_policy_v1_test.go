package miningtransition

import "testing"

func TestDefaultMiningTransitionPolicyValid(t *testing.T) {
	policy := DefaultPolicyV1()

	if err := ValidatePolicyV1(policy); err != nil {
		t.Fatalf("default mining transition policy should validate: %v", err)
	}
}

func TestPrimaryMiningRewardSchedule(t *testing.T) {
	policy := DefaultPolicyV1()

	expectedRewards := []string{"50", "25", "12.5", "6.25", "3.125"}
	if len(policy.PrimaryRewardSchedule) != len(expectedRewards) {
		t.Fatalf("primary reward schedule length mismatch")
	}
	for i, reward := range expectedRewards {
		if policy.PrimaryRewardSchedule[i].RewardQC != reward {
			t.Fatalf("primary reward mismatch at period %d", i)
		}
	}
}

func TestBitcoinStyleContinuationSchedule(t *testing.T) {
	policy := DefaultPolicyV1()

	expectedRewards := []string{"1.5625", "0.78125", "0.390625"}
	for i, reward := range expectedRewards {
		if policy.PostTenYearExampleSchedule[i].RewardQC != reward {
			t.Fatalf("post-ten-year reward mismatch at period %d", i)
		}
	}
}

func TestTenYearTwoYearHalvingInvariant(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.PrimaryMiningPeriodYears != 10 {
		t.Fatalf("primary mining period must be 10 years")
	}
	if policy.HalvingIntervalYears != 2 {
		t.Fatalf("halving interval must be 2 years")
	}
}

func TestIncompleteAfterTenYearsWalletMinerContinuesBitcoinStyle(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.IfNotFullyMinedAfterPrimaryPeriod != "WALLET_MINER_CONTINUES_WITH_BITCOIN_STYLE_HALVING_UNTIL_MAX_SUPPLY" {
		t.Fatalf("incomplete emission after 10 years must continue wallet+miner with bitcoin-style halving")
	}
	if !policy.WalletMinerContinuesIfIncomplete {
		t.Fatalf("wallet+miner must continue if supply remains")
	}
	if !policy.BitcoinStyleHalvingContinues {
		t.Fatalf("bitcoin-style halving must continue after 10 years")
	}
	if !policy.MiningContinuesOnlyUntilMaxSupply {
		t.Fatalf("continued mining must stop at max supply")
	}
	if !policy.RewardMustBeCappedByRemainingSupply {
		t.Fatalf("reward must be capped by remaining supply")
	}
	if !policy.NodeFeeOnlyNotForcedWhileSupplyRemains {
		t.Fatalf("node fee-only mode must not be forced while supply remains")
	}
}

func TestRejectAutoMintRemaining(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.AutoMintRemainingForbidden = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("auto mint remaining must fail")
	}
}

func TestRejectNodeFeeOnlyWhileSupplyRemains(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.NodeFeeOnlyNotForcedWhileSupplyRemains = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("forcing node fee-only mode while supply remains must fail")
	}
}
