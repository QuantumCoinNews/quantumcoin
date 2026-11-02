// Package miningtransition contains QuantumCoin project code for this module.
package miningtransition

import (
	"errors"
	"strconv"
	"strings"

	qconsensus "quantumcoin/consensus"
)

const MiningNodeTransitionVersionV1 = "QC_MINING_PERIOD_BITCOIN_STYLE_CONTINUE_V1"

const (
	MaxSupplyQCV1                int64 = qconsensus.CanonicalTotalSupplyQCV1
	InitialRewardQCV1            int64 = qconsensus.CanonicalInitialMiningRewardQCV1
	PrimaryMiningPeriodYearsV1   int   = 10
	HalvingIntervalYearsV1       int   = 2
	ExpectedPrimaryPeriodCountV1 int   = 5
)

type RewardPeriod struct {
	Index    int    `json:"index"`
	FromYear int    `json:"fromYear"`
	ToYear   int    `json:"toYear"`
	RewardQC string `json:"rewardQc"`
}

type MiningNodeTransitionPolicy struct {
	Version                                string          `json:"version"`
	Mode                                   string          `json:"mode"`
	MaxSupplyQC                            int64           `json:"maxSupplyQc"`
	InitialRewardQC                        int64           `json:"initialRewardQc"`
	PrimaryMiningPeriodYears               int             `json:"primaryMiningPeriodYears"`
	HalvingIntervalYears                   int             `json:"halvingIntervalYears"`
	PrimaryRewardSchedule                  []RewardPeriod  `json:"primaryRewardSchedule"`
	PostTenYearExampleSchedule             []RewardPeriod  `json:"postTenYearExampleSchedule"`
	IfFullyMinedWithinPrimaryPeriod        string          `json:"ifFullyMinedWithinPrimaryPeriod"`
	IfNotFullyMinedAfterPrimaryPeriod      string          `json:"ifNotFullyMinedAfterPrimaryPeriod"`
	WalletMinerContinuesIfIncomplete       bool            `json:"walletMinerContinuesIfIncomplete"`
	BitcoinStyleHalvingContinues           bool            `json:"bitcoinStyleHalvingContinues"`
	MiningContinuesOnlyUntilMaxSupply      bool            `json:"miningContinuesOnlyUntilMaxSupply"`
	RewardMustBeCappedByRemainingSupply    bool            `json:"rewardMustBeCappedByRemainingSupply"`
	AutoMintRemainingForbidden             bool            `json:"autoMintRemainingForbidden"`
	AdminMintRemainingForbidden            bool            `json:"adminMintRemainingForbidden"`
	AICannotDistributeRemaining            bool            `json:"aiCannotDistributeRemaining"`
	GameCannotDistributeRemaining          bool            `json:"gameCannotDistributeRemaining"`
	WorkerCannotDistributeRemaining        bool            `json:"workerCannotDistributeRemaining"`
	NodeFeeModeEnabledAfterEmissionDone    bool            `json:"nodeFeeModeEnabledAfterEmissionDone"`
	NodeFeeOnlyNotForcedWhileSupplyRemains bool            `json:"nodeFeeOnlyNotForcedWhileSupplyRemains"`
	MaxSupplyAlwaysUnchanged               bool            `json:"maxSupplyAlwaysUnchanged"`
	SafeBoundary                           map[string]bool `json:"safeBoundary"`
}

func rewardForHalvingV1(halvings int) string {
	if halvings < 0 {
		return ""
	}

	numerator := InitialRewardQCV1
	denominator := int64(1) << halvings

	whole := numerator / denominator
	remainder := numerator % denominator

	if remainder == 0 {
		return strconv.FormatInt(whole, 10)
	}

	// denominator is always a power of two, so the decimal expansion
	// terminates exactly. Convert the rational value without changing
	// the canonical initial reward authority.
	value := float64(numerator) / float64(denominator)

	return strconv.FormatFloat(value, 'f', -1, 64)
}
func PrimaryRewardScheduleV1() []RewardPeriod {
	return []RewardPeriod{
		{Index: 1, FromYear: 0, ToYear: 2, RewardQC: rewardForHalvingV1(0)},
		{Index: 2, FromYear: 2, ToYear: 4, RewardQC: rewardForHalvingV1(1)},
		{Index: 3, FromYear: 4, ToYear: 6, RewardQC: rewardForHalvingV1(2)},
		{Index: 4, FromYear: 6, ToYear: 8, RewardQC: rewardForHalvingV1(3)},
		{Index: 5, FromYear: 8, ToYear: 10, RewardQC: rewardForHalvingV1(4)},
	}
}

func PostTenYearExampleScheduleV1() []RewardPeriod {
	return []RewardPeriod{
		{Index: 6, FromYear: 10, ToYear: 12, RewardQC: rewardForHalvingV1(5)},
		{Index: 7, FromYear: 12, ToYear: 14, RewardQC: rewardForHalvingV1(6)},
		{Index: 8, FromYear: 14, ToYear: 16, RewardQC: rewardForHalvingV1(7)},
	}
}

func DefaultPolicyV1() MiningNodeTransitionPolicy {
	return MiningNodeTransitionPolicy{
		Version:                                MiningNodeTransitionVersionV1,
		Mode:                                   "MINING_PERIOD_BITCOIN_STYLE_CONTINUE_NO_MINT_EXECUTION",
		MaxSupplyQC:                            MaxSupplyQCV1,
		InitialRewardQC:                        InitialRewardQCV1,
		PrimaryMiningPeriodYears:               PrimaryMiningPeriodYearsV1,
		HalvingIntervalYears:                   HalvingIntervalYearsV1,
		PrimaryRewardSchedule:                  PrimaryRewardScheduleV1(),
		PostTenYearExampleSchedule:             PostTenYearExampleScheduleV1(),
		IfFullyMinedWithinPrimaryPeriod:        "NODE_FEE_MODE_AFTER_EMISSION_COMPLETE",
		IfNotFullyMinedAfterPrimaryPeriod:      "WALLET_MINER_CONTINUES_WITH_BITCOIN_STYLE_HALVING_UNTIL_MAX_SUPPLY",
		WalletMinerContinuesIfIncomplete:       true,
		BitcoinStyleHalvingContinues:           true,
		MiningContinuesOnlyUntilMaxSupply:      true,
		RewardMustBeCappedByRemainingSupply:    true,
		AutoMintRemainingForbidden:             true,
		AdminMintRemainingForbidden:            true,
		AICannotDistributeRemaining:            true,
		GameCannotDistributeRemaining:          true,
		WorkerCannotDistributeRemaining:        true,
		NodeFeeModeEnabledAfterEmissionDone:    true,
		NodeFeeOnlyNotForcedWhileSupplyRemains: true,
		MaxSupplyAlwaysUnchanged:               true,
		SafeBoundary:                           SafeBoundaryV1(),
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotMint":                              true,
		"doesNotAutoMintRemaining":                 true,
		"doesNotAdminMintRemaining":                true,
		"doesNotAIDistributeRemaining":             true,
		"doesNotGameDistributeRemaining":           true,
		"doesNotWorkerDistributeRemaining":         true,
		"doesNotChangeMaxSupply":                   true,
		"doesNotChangeBalances":                    true,
		"doesNotChangeChain":                       true,
		"doesNotChangeConsensus":                   true,
		"doesNotChangeMiningRewardNow":             true,
		"doesNotStartMining":                       true,
		"doesNotStopMining":                        true,
		"requiresTenYearPrimaryMiningPeriod":       true,
		"requiresTwoYearHalving":                   true,
		"requiresWalletMinerContinueIfIncomplete":  true,
		"requiresBitcoinStyleHalvingAfterTenYears": true,
		"requiresRewardCapByRemainingSupply":       true,
		"requiresNodeFeeModeAfterEmissionDone":     true,
		"forbidsNodeFeeOnlyWhileSupplyRemains":     true,
		"controlOnly":                              true,
	}
}

func ValidatePolicyV1(policy MiningNodeTransitionPolicy) error {
	if strings.TrimSpace(policy.Version) != MiningNodeTransitionVersionV1 {
		return errors.New("invalid mining transition version")
	}
	if strings.TrimSpace(policy.Mode) != "MINING_PERIOD_BITCOIN_STYLE_CONTINUE_NO_MINT_EXECUTION" {
		return errors.New("invalid mining transition mode")
	}
	if policy.MaxSupplyQC != MaxSupplyQCV1 {
		return errors.New("max supply must remain 25,500,000 QC")
	}
	if policy.InitialRewardQC != InitialRewardQCV1 {
		return errors.New("initial reward must be 50 QC")
	}
	if policy.PrimaryMiningPeriodYears != PrimaryMiningPeriodYearsV1 {
		return errors.New("primary mining period must be 10 years")
	}
	if policy.HalvingIntervalYears != HalvingIntervalYearsV1 {
		return errors.New("halving interval must be 2 years")
	}
	if len(policy.PrimaryRewardSchedule) != ExpectedPrimaryPeriodCountV1 {
		return errors.New("primary reward schedule must contain 5 periods")
	}

	expected := PrimaryRewardScheduleV1()
	for i := range expected {
		got := policy.PrimaryRewardSchedule[i]
		want := expected[i]
		if got.Index != want.Index || got.FromYear != want.FromYear || got.ToYear != want.ToYear || got.RewardQC != want.RewardQC {
			return errors.New("invalid primary reward schedule")
		}
	}

	post := policy.PostTenYearExampleSchedule
	expectedPost := PostTenYearExampleScheduleV1()
	if len(post) != len(expectedPost) {
		return errors.New("invalid post-ten-year bitcoin-style example schedule")
	}

	for i := range expectedPost {
		got := post[i]
		want := expectedPost[i]

		if got.Index != want.Index ||
			got.FromYear != want.FromYear ||
			got.ToYear != want.ToYear ||
			got.RewardQC != want.RewardQC {
			return errors.New("invalid post-ten-year bitcoin-style example schedule")
		}
	}
	if policy.IfFullyMinedWithinPrimaryPeriod != "NODE_FEE_MODE_AFTER_EMISSION_COMPLETE" {
		return errors.New("fully mined transition must go to node fee mode")
	}
	if policy.IfNotFullyMinedAfterPrimaryPeriod != "WALLET_MINER_CONTINUES_WITH_BITCOIN_STYLE_HALVING_UNTIL_MAX_SUPPLY" {
		return errors.New("incomplete mining after 10 years must continue wallet+miner with bitcoin-style halving")
	}
	if !policy.WalletMinerContinuesIfIncomplete {
		return errors.New("wallet+miner must continue if supply is incomplete after 10 years")
	}
	if !policy.BitcoinStyleHalvingContinues {
		return errors.New("bitcoin-style halving must continue after 10 years if supply remains")
	}
	if !policy.MiningContinuesOnlyUntilMaxSupply {
		return errors.New("continued mining must stop at max supply")
	}
	if !policy.RewardMustBeCappedByRemainingSupply {
		return errors.New("reward must be capped by remaining supply")
	}
	if !policy.AutoMintRemainingForbidden {
		return errors.New("auto mint of remaining coins must be forbidden")
	}
	if !policy.AdminMintRemainingForbidden {
		return errors.New("admin mint of remaining coins must be forbidden")
	}
	if !policy.AICannotDistributeRemaining {
		return errors.New("AI distribution of remaining coins must be forbidden")
	}
	if !policy.GameCannotDistributeRemaining {
		return errors.New("game distribution of remaining coins must be forbidden")
	}
	if !policy.WorkerCannotDistributeRemaining {
		return errors.New("worker distribution of remaining coins must be forbidden")
	}
	if !policy.NodeFeeModeEnabledAfterEmissionDone {
		return errors.New("node fee mode must be enabled after emission completes")
	}
	if !policy.NodeFeeOnlyNotForcedWhileSupplyRemains {
		return errors.New("node fee-only mode must not be forced while supply remains")
	}
	if !policy.MaxSupplyAlwaysUnchanged {
		return errors.New("max supply must always remain unchanged")
	}
	if err := validateExactSafeBoundaryMapV1(
		policy.SafeBoundary,
		SafeBoundaryV1(),
	); err != nil {
		return err
	}

	for _, k := range []string{
		"doesNotMint",
		"doesNotAutoMintRemaining",
		"doesNotAdminMintRemaining",
		"doesNotAIDistributeRemaining",
		"doesNotGameDistributeRemaining",
		"doesNotWorkerDistributeRemaining",
		"doesNotChangeMaxSupply",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeConsensus",
		"doesNotChangeMiningRewardNow",
		"doesNotStartMining",
		"doesNotStopMining",
		"requiresTenYearPrimaryMiningPeriod",
		"requiresTwoYearHalving",
		"requiresWalletMinerContinueIfIncomplete",
		"requiresBitcoinStyleHalvingAfterTenYears",
		"requiresRewardCapByRemainingSupply",
		"requiresNodeFeeModeAfterEmissionDone",
		"forbidsNodeFeeOnlyWhileSupplyRemains",
		"controlOnly",
	} {
		if !policy.SafeBoundary[k] {
			return errors.New("safe boundary missing: " + k)
		}
	}

	return nil
}

func validateExactSafeBoundaryMapV1(
	boundary map[string]bool,
	expected map[string]bool,
) error {
	if boundary == nil {
		return errors.New("mining transition policy safe boundary required")
	}

	if len(boundary) != len(expected) {
		return errors.New("mining transition policy safe boundary key count mismatch")
	}

	for key, want := range expected {
		got, exists := boundary[key]
		if !exists {
			return errors.New("safe boundary missing: " + key)
		}
		if got != want {
			return errors.New("safe boundary value mismatch: " + key)
		}
	}

	for key := range boundary {
		if _, exists := expected[key]; !exists {
			return errors.New("unexpected safe boundary key: " + key)
		}
	}

	return nil
}
