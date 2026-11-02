package consensus

import (
	"fmt"
	"strings"
)

const CanonicalEconomyPolicyVersionV1 = "QC_CANONICAL_ECONOMY_POLICY_V1"

const (
	CanonicalFullAllocationBasisPointsV1 = 10_000

	CanonicalFounderTotalBasisPointsV1  = 500
	CanonicalFounderLiquidBasisPointsV1 = 100
	CanonicalFounderLockedBasisPointsV1 = 400
	CanonicalNonFounderBasisPointsV1    = 9_500

	CanonicalTotalSupplyQCV1          int64 = 25_500_000
	CanonicalInitialMiningRewardQCV1  int64 = 50
	CanonicalCoinbaseMaturityBlocksV1       = 10

	CanonicalFounderTotalQCV1 int64 = CanonicalTotalSupplyQCV1 *
		CanonicalFounderTotalBasisPointsV1 /
		CanonicalFullAllocationBasisPointsV1

	CanonicalFounderLiquidQCV1 int64 = CanonicalTotalSupplyQCV1 *
		CanonicalFounderLiquidBasisPointsV1 /
		CanonicalFullAllocationBasisPointsV1

	CanonicalFounderLockedQCV1 int64 = CanonicalTotalSupplyQCV1 *
		CanonicalFounderLockedBasisPointsV1 /
		CanonicalFullAllocationBasisPointsV1

	CanonicalFounderCliffMonthsV1   = 12
	CanonicalFounderVestingMonthsV1 = 48

	CanonicalFounderScheduleMonthsV1 = CanonicalFounderCliffMonthsV1 +
		CanonicalFounderVestingMonthsV1

	CanonicalFounderUnlockStartMonthV1 = CanonicalFounderCliffMonthsV1 + 1

	CanonicalFounderUnlockEndMonthV1 = CanonicalFounderScheduleMonthsV1

	CanonicalFounderMonthlyUnlockQCV1 int64 = CanonicalFounderLockedQCV1 /
		CanonicalFounderVestingMonthsV1

	CanonicalFounderTotalPercentV1        = 5
	CanonicalFounderLiquidPercentV1       = 1
	CanonicalFounderLockedPercentV1       = 4
	CanonicalMiningEmissionPoolQCV1 int64 = CanonicalTotalSupplyQCV1 -
		CanonicalFounderTotalQCV1

	CanonicalMiningEmissionPoolAddressV1 = "QC_MINING_EMISSION_POOL"
	CanonicalBurnSinkAddressV1           = "QC_BURN_SINK"
)

// ValidateCanonicalEconomyPolicyV1 validates the one authoritative monetary
// policy definition used by config, blockchain and runtime policy adapters.
func ValidateCanonicalEconomyPolicyV1() error {
	if CanonicalTotalSupplyQCV1 <= 0 {
		return fmt.Errorf(
			"%s: total supply must be positive",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	if CanonicalInitialMiningRewardQCV1 <= 0 ||
		CanonicalInitialMiningRewardQCV1 >
			CanonicalTotalSupplyQCV1 {
		return fmt.Errorf(
			"%s: invalid initial mining reward",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	if CanonicalCoinbaseMaturityBlocksV1 <= 0 {
		return fmt.Errorf(
			"%s: coinbase maturity must be positive",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	if CanonicalFounderLiquidBasisPointsV1+
		CanonicalFounderLockedBasisPointsV1 !=
		CanonicalFounderTotalBasisPointsV1 {
		return fmt.Errorf(
			"%s: founder basis-point split mismatch",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	if CanonicalFounderTotalBasisPointsV1+
		CanonicalNonFounderBasisPointsV1 !=
		CanonicalFullAllocationBasisPointsV1 {
		return fmt.Errorf(
			"%s: allocation basis points do not total 100%%",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	if CanonicalFounderLiquidQCV1+
		CanonicalFounderLockedQCV1 !=
		CanonicalFounderTotalQCV1 {
		return fmt.Errorf(
			"%s: founder amount split mismatch",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	if CanonicalFounderVestingMonthsV1 <= 0 ||
		CanonicalFounderLockedQCV1%
			CanonicalFounderVestingMonthsV1 != 0 {
		return fmt.Errorf(
			"%s: founder locked allocation is not evenly vestable",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	if CanonicalFounderMonthlyUnlockQCV1*
		CanonicalFounderVestingMonthsV1 !=
		CanonicalFounderLockedQCV1 {
		return fmt.Errorf(
			"%s: founder monthly unlock mismatch",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	if CanonicalFounderUnlockStartMonthV1 !=
		CanonicalFounderCliffMonthsV1+1 ||
		CanonicalFounderUnlockEndMonthV1 !=
			CanonicalFounderScheduleMonthsV1 {
		return fmt.Errorf(
			"%s: founder unlock schedule mismatch",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	if CanonicalMiningEmissionPoolQCV1 <= 0 {
		return fmt.Errorf(
			"%s: mining emission pool must be positive",
			CanonicalEconomyPolicyVersionV1,
		)
	}

	addresses := []string{
		CanonicalMiningEmissionPoolAddressV1,
		CanonicalBurnSinkAddressV1,
	}

	seen := make(map[string]struct{}, len(addresses))

	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if address == "" {
			return fmt.Errorf(
				"%s: canonical economy address required",
				CanonicalEconomyPolicyVersionV1,
			)
		}

		if _, exists := seen[address]; exists {
			return fmt.Errorf(
				"%s: duplicate canonical economy address: %s",
				CanonicalEconomyPolicyVersionV1,
				address,
			)
		}

		seen[address] = struct{}{}
	}

	return nil
}
