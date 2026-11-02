package blockchain

import (
	"fmt"
	"math"
)

const BurnNetworkLifetimePolicyVersionV1 = "QC_BURN_NETWORK_LIFETIME_POLICY_V1"

func (bc *Blockchain) validateCanonicalBurnNetworkLifetimeLimitAtHeightV1(
	tx *Transaction,
	spendHeight int,
	candidatePriorBurnAtoms int64,
	maxAtoms int64,
) error {
	if bc == nil {
		return fmt.Errorf(
			"%s: nil blockchain",
			BurnNetworkLifetimePolicyVersionV1,
		)
	}

	if tx == nil {
		return fmt.Errorf(
			"%s: nil transaction",
			BurnNetworkLifetimePolicyVersionV1,
		)
	}

	if spendHeight < 0 {
		return fmt.Errorf(
			"%s: invalid spend height %d",
			BurnNetworkLifetimePolicyVersionV1,
			spendHeight,
		)
	}

	if candidatePriorBurnAtoms < 0 {
		return fmt.Errorf(
			"%s: negative candidate prior burn: %d atoms",
			BurnNetworkLifetimePolicyVersionV1,
			candidatePriorBurnAtoms,
		)
	}

	if maxAtoms <= 0 {
		return fmt.Errorf(
			"%s: invalid network lifetime burn limit: %d atoms",
			BurnNetworkLifetimePolicyVersionV1,
			maxAtoms,
		)
	}

	classification, err :=
		ClassifyCanonicalBurnTransactionAtHeightV1(
			tx,
			spendHeight,
			bc,
		)
	if err != nil {
		return fmt.Errorf(
			"%s: classify transaction: %w",
			BurnNetworkLifetimePolicyVersionV1,
			err,
		)
	}

	if !classification.IsBurn {
		return nil
	}

	if classification.BurnAmountAtoms <= 0 {
		return fmt.Errorf(
			"%s: invalid canonical burn amount: %d atoms",
			BurnNetworkLifetimePolicyVersionV1,
			classification.BurnAmountAtoms,
		)
	}

	historicalBurnAtoms, err :=
		bc.TotalBurnedAtomsCheckedV1()
	if err != nil {
		return fmt.Errorf(
			"%s: historical network burn accounting: %w",
			BurnNetworkLifetimePolicyVersionV1,
			err,
		)
	}

	if historicalBurnAtoms < 0 {
		return fmt.Errorf(
			"%s: negative historical network burn: %d atoms",
			BurnNetworkLifetimePolicyVersionV1,
			historicalBurnAtoms,
		)
	}

	if historicalBurnAtoms >
		math.MaxInt64-candidatePriorBurnAtoms {
		return fmt.Errorf(
			"%s: network lifetime burn overflow before candidate transaction",
			BurnNetworkLifetimePolicyVersionV1,
		)
	}

	totalBeforeCurrent :=
		historicalBurnAtoms +
			candidatePriorBurnAtoms

	if totalBeforeCurrent >
		math.MaxInt64-classification.BurnAmountAtoms {
		return fmt.Errorf(
			"%s: network lifetime burn overflow",
			BurnNetworkLifetimePolicyVersionV1,
		)
	}

	total :=
		totalBeforeCurrent +
			classification.BurnAmountAtoms

	if total > maxAtoms {
		return fmt.Errorf(
			"%s: network lifetime burn limit exceeded: historical=%d candidate_prior=%d current=%d total=%d max=%d atoms",
			BurnNetworkLifetimePolicyVersionV1,
			historicalBurnAtoms,
			candidatePriorBurnAtoms,
			classification.BurnAmountAtoms,
			total,
			maxAtoms,
		)
	}

	return nil
}
