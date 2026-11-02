// Package foundervesting contains QuantumCoin project code for this module.
package foundervesting

import (
	"errors"
	"strings"

	qconsensus "quantumcoin/consensus"
)

const FounderVestingPolicyVersionV1 = "QC_FOUNDER_VESTING_POLICY_V1"

const (
	MaxSupplyQCV1          int64 = qconsensus.CanonicalTotalSupplyQCV1
	FounderTotalQCV1       int64 = qconsensus.CanonicalFounderTotalQCV1
	FounderLiquidQCV1      int64 = qconsensus.CanonicalFounderLiquidQCV1
	FounderLockedQCV1      int64 = qconsensus.CanonicalFounderLockedQCV1
	FounderTotalPercentV1  int64 = qconsensus.CanonicalFounderTotalPercentV1
	FounderLiquidPercentV1 int64 = qconsensus.CanonicalFounderLiquidPercentV1
	FounderLockedPercentV1 int64 = qconsensus.CanonicalFounderLockedPercentV1
	CliffMonthsV1          int   = qconsensus.CanonicalFounderCliffMonthsV1
	VestingMonthsV1        int   = qconsensus.CanonicalFounderVestingMonthsV1
	TotalScheduleMonthsV1  int   = qconsensus.CanonicalFounderScheduleMonthsV1
	MonthlyUnlockQCV1      int64 = qconsensus.CanonicalFounderMonthlyUnlockQCV1
)

type FounderVestingPolicy struct {
	Version                              string          `json:"version"`
	Mode                                 string          `json:"mode"`
	MaxSupplyQC                          int64           `json:"maxSupplyQc"`
	FounderTotalQC                       int64           `json:"founderTotalQc"`
	FounderLiquidQC                      int64           `json:"founderLiquidQc"`
	FounderLockedQC                      int64           `json:"founderLockedQc"`
	FounderTotalPercent                  int64           `json:"founderTotalPercent"`
	FounderLiquidPercent                 int64           `json:"founderLiquidPercent"`
	FounderLockedPercent                 int64           `json:"founderLockedPercent"`
	CliffMonths                          int             `json:"cliffMonths"`
	VestingMonths                        int             `json:"vestingMonths"`
	TotalScheduleMonths                  int             `json:"totalScheduleMonths"`
	MonthlyUnlockQC                      int64           `json:"monthlyUnlockQc"`
	UnlockStartsAtMonth                  int             `json:"unlockStartsAtMonth"`
	UnlockEndsAtMonth                    int             `json:"unlockEndsAtMonth"`
	FounderWalletFinalBeforeMainnet      bool            `json:"founderWalletFinalBeforeMainnet"`
	FounderWalletChangeAfterMainnet      bool            `json:"founderWalletChangeAfterMainnet"`
	EarlyUnlockForbidden                 bool            `json:"earlyUnlockForbidden"`
	AdminEarlyUnlockForbidden            bool            `json:"adminEarlyUnlockForbidden"`
	AICannotUnlock                       bool            `json:"aiCannotUnlock"`
	GameCannotUnlock                     bool            `json:"gameCannotUnlock"`
	WorkerCannotUnlock                   bool            `json:"workerCannotUnlock"`
	TGWTCannotUnlock                     bool            `json:"tgwtCannotUnlock"`
	NFTCannotUnlock                      bool            `json:"nftCannotUnlock"`
	DoesNotMintFounderAllocation         bool            `json:"doesNotMintFounderAllocation"`
	FounderAllocationFromFixedSupplyOnly bool            `json:"founderAllocationFromFixedSupplyOnly"`
	MaxSupplyAlwaysUnchanged             bool            `json:"maxSupplyAlwaysUnchanged"`
	SafeBoundary                         map[string]bool `json:"safeBoundary"`
}

func DefaultPolicyV1() FounderVestingPolicy {
	return FounderVestingPolicy{
		Version:                              FounderVestingPolicyVersionV1,
		Mode:                                 "FOUNDER_VESTING_LOCK_POLICY_NO_UNLOCK_EXECUTION",
		MaxSupplyQC:                          MaxSupplyQCV1,
		FounderTotalQC:                       FounderTotalQCV1,
		FounderLiquidQC:                      FounderLiquidQCV1,
		FounderLockedQC:                      FounderLockedQCV1,
		FounderTotalPercent:                  FounderTotalPercentV1,
		FounderLiquidPercent:                 FounderLiquidPercentV1,
		FounderLockedPercent:                 FounderLockedPercentV1,
		CliffMonths:                          CliffMonthsV1,
		VestingMonths:                        VestingMonthsV1,
		TotalScheduleMonths:                  TotalScheduleMonthsV1,
		MonthlyUnlockQC:                      MonthlyUnlockQCV1,
		UnlockStartsAtMonth:                  qconsensus.CanonicalFounderUnlockStartMonthV1,
		UnlockEndsAtMonth:                    qconsensus.CanonicalFounderUnlockEndMonthV1,
		FounderWalletFinalBeforeMainnet:      true,
		FounderWalletChangeAfterMainnet:      false,
		EarlyUnlockForbidden:                 true,
		AdminEarlyUnlockForbidden:            true,
		AICannotUnlock:                       true,
		GameCannotUnlock:                     true,
		WorkerCannotUnlock:                   true,
		TGWTCannotUnlock:                     true,
		NFTCannotUnlock:                      true,
		DoesNotMintFounderAllocation:         true,
		FounderAllocationFromFixedSupplyOnly: true,
		MaxSupplyAlwaysUnchanged:             true,
		SafeBoundary:                         SafeBoundaryV1(),
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotMint":                             true,
		"doesNotChangeMaxSupply":                  true,
		"doesNotChangeBalances":                   true,
		"doesNotSendFunds":                        true,
		"doesNotExecuteUnlock":                    true,
		"doesNotChangeConsensus":                  true,
		"requiresFounderWalletFinalBeforeMainnet": true,
		"forbidsFounderWalletChangeAfterMainnet":  true,
		"requiresOneYearCliff":                    true,
		"requiresFourYearMonthlyVesting":          true,
		"requiresFortyEightMonthlyUnlocks":        true,
		"requiresMonthlyUnlock21250QC":            true,
		"forbidsEarlyUnlock":                      true,
		"adminCannotEarlyUnlock":                  true,
		"aiCannotUnlock":                          true,
		"gameCannotUnlock":                        true,
		"workerCannotUnlock":                      true,
		"tgwtCannotUnlock":                        true,
		"nftCannotUnlock":                         true,
		"fixedSupplyOnly":                         true,
		"controlOnly":                             true,
	}
}

func ValidatePolicyV1(policy FounderVestingPolicy) error {
	if strings.TrimSpace(policy.Version) != FounderVestingPolicyVersionV1 {
		return errors.New("invalid founder vesting policy version")
	}
	if strings.TrimSpace(policy.Mode) != "FOUNDER_VESTING_LOCK_POLICY_NO_UNLOCK_EXECUTION" {
		return errors.New("invalid founder vesting policy mode")
	}
	if policy.MaxSupplyQC != MaxSupplyQCV1 {
		return errors.New("max supply must remain 25,500,000 QC")
	}
	if policy.FounderTotalQC != FounderTotalQCV1 {
		return errors.New("founder total allocation must be 1,275,000 QC")
	}
	if policy.FounderLiquidQC != FounderLiquidQCV1 {
		return errors.New("founder liquid allocation must be 255,000 QC")
	}
	if policy.FounderLockedQC != FounderLockedQCV1 {
		return errors.New("founder locked allocation must be 1,020,000 QC")
	}
	if policy.FounderTotalPercent != FounderTotalPercentV1 || policy.FounderLiquidPercent != FounderLiquidPercentV1 || policy.FounderLockedPercent != FounderLockedPercentV1 {
		return errors.New("founder allocation percentages must be 5 total, 1 liquid, 4 locked")
	}
	if policy.CliffMonths != CliffMonthsV1 {
		return errors.New("founder cliff must be 12 months")
	}
	if policy.VestingMonths != VestingMonthsV1 {
		return errors.New("founder vesting must be 48 months")
	}
	if policy.TotalScheduleMonths != TotalScheduleMonthsV1 {
		return errors.New("founder schedule must be 60 months total")
	}
	if policy.MonthlyUnlockQC != MonthlyUnlockQCV1 {
		return errors.New("monthly unlock must be 21,250 QC")
	}
	if policy.MonthlyUnlockQC*int64(policy.VestingMonths) != policy.FounderLockedQC {
		return errors.New("monthly unlock times vesting months must equal locked allocation")
	}
	if policy.UnlockStartsAtMonth != qconsensus.CanonicalFounderUnlockStartMonthV1 ||
		policy.UnlockEndsAtMonth != qconsensus.CanonicalFounderUnlockEndMonthV1 {
		return errors.New("unlock months must be 13 through 60")
	}
	if !policy.FounderWalletFinalBeforeMainnet {
		return errors.New("founder wallet must be final before mainnet")
	}
	if policy.FounderWalletChangeAfterMainnet {
		return errors.New("founder wallet must not change after mainnet")
	}
	if !policy.EarlyUnlockForbidden || !policy.AdminEarlyUnlockForbidden {
		return errors.New("early unlock must be forbidden")
	}
	if !policy.AICannotUnlock || !policy.GameCannotUnlock || !policy.WorkerCannotUnlock || !policy.TGWTCannotUnlock || !policy.NFTCannotUnlock {
		return errors.New("automation and modules must not unlock founder funds")
	}
	if !policy.DoesNotMintFounderAllocation || !policy.FounderAllocationFromFixedSupplyOnly || !policy.MaxSupplyAlwaysUnchanged {
		return errors.New("founder allocation must come only from fixed supply")
	}
	if err := validateExactSafeBoundaryMapV1(
		policy.SafeBoundary,
		SafeBoundaryV1(),
	); err != nil {
		return err
	}

	for _, k := range []string{
		"doesNotMint",
		"doesNotChangeMaxSupply",
		"doesNotChangeBalances",
		"doesNotSendFunds",
		"doesNotExecuteUnlock",
		"doesNotChangeConsensus",
		"requiresFounderWalletFinalBeforeMainnet",
		"forbidsFounderWalletChangeAfterMainnet",
		"requiresOneYearCliff",
		"requiresFourYearMonthlyVesting",
		"requiresFortyEightMonthlyUnlocks",
		"requiresMonthlyUnlock21250QC",
		"forbidsEarlyUnlock",
		"adminCannotEarlyUnlock",
		"aiCannotUnlock",
		"gameCannotUnlock",
		"workerCannotUnlock",
		"tgwtCannotUnlock",
		"nftCannotUnlock",
		"fixedSupplyOnly",
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
		return errors.New("founder vesting policy safe boundary required")
	}

	if len(boundary) != len(expected) {
		return errors.New("founder vesting policy safe boundary key count mismatch")
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
