package blockchain

import (
	"bytes"
	"fmt"
	"math"
)

const BurnOwnerLifetimePolicyVersionV1 = "QC_BURN_OWNER_LIFETIME_POLICY_V1"

func (bc *Blockchain) validateCanonicalBurnOwnerLifetimeLimitAtHeightV1(
	tx *Transaction,
	spendHeight int,
	ownerKey []byte,
	candidatePriorBurnAtoms int64,
	maxAtoms int64,
) error {
	if bc == nil {
		return fmt.Errorf(
			"%s: nil blockchain",
			BurnOwnerLifetimePolicyVersionV1,
		)
	}

	if tx == nil {
		return fmt.Errorf(
			"%s: nil transaction",
			BurnOwnerLifetimePolicyVersionV1,
		)
	}

	if spendHeight < 0 {
		return fmt.Errorf(
			"%s: invalid spend height %d",
			BurnOwnerLifetimePolicyVersionV1,
			spendHeight,
		)
	}

	if len(ownerKey) == 0 {
		return fmt.Errorf(
			"%s: empty owner key",
			BurnOwnerLifetimePolicyVersionV1,
		)
	}

	if candidatePriorBurnAtoms < 0 {
		return fmt.Errorf(
			"%s: negative candidate prior burn: %d atoms",
			BurnOwnerLifetimePolicyVersionV1,
			candidatePriorBurnAtoms,
		)
	}

	if maxAtoms <= 0 {
		return fmt.Errorf(
			"%s: invalid owner lifetime burn limit: %d atoms",
			BurnOwnerLifetimePolicyVersionV1,
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
			BurnOwnerLifetimePolicyVersionV1,
			err,
		)
	}

	if !classification.IsBurn {
		return nil
	}

	if classification.BurnAmountAtoms <= 0 {
		return fmt.Errorf(
			"%s: invalid canonical burn amount: %d atoms",
			BurnOwnerLifetimePolicyVersionV1,
			classification.BurnAmountAtoms,
		)
	}

	txOwnerKey, err :=
		ResolveValidatedHistoricalBurnOwnerKeyV1(
			tx,
		)
	if err != nil {
		return fmt.Errorf(
			"%s: resolve transaction owner: %w",
			BurnOwnerLifetimePolicyVersionV1,
			err,
		)
	}

	if !bytes.Equal(
		txOwnerKey,
		ownerKey,
	) {
		return fmt.Errorf(
			"%s: supplied owner does not match transaction inputs",
			BurnOwnerLifetimePolicyVersionV1,
		)
	}

	historicalBurnAtoms, err :=
		bc.TotalBurnedAtomsByOwnerCheckedV1(
			ownerKey,
		)
	if err != nil {
		return fmt.Errorf(
			"%s: historical owner burn accounting: %w",
			BurnOwnerLifetimePolicyVersionV1,
			err,
		)
	}

	if historicalBurnAtoms < 0 {
		return fmt.Errorf(
			"%s: negative historical owner burn: %d atoms",
			BurnOwnerLifetimePolicyVersionV1,
			historicalBurnAtoms,
		)
	}

	if historicalBurnAtoms >
		math.MaxInt64-candidatePriorBurnAtoms {
		return fmt.Errorf(
			"%s: owner lifetime burn overflow before candidate transaction",
			BurnOwnerLifetimePolicyVersionV1,
		)
	}

	totalBeforeCurrent :=
		historicalBurnAtoms +
			candidatePriorBurnAtoms

	if totalBeforeCurrent >
		math.MaxInt64-classification.BurnAmountAtoms {
		return fmt.Errorf(
			"%s: owner lifetime burn overflow",
			BurnOwnerLifetimePolicyVersionV1,
		)
	}

	total :=
		totalBeforeCurrent +
			classification.BurnAmountAtoms

	if total > maxAtoms {
		return fmt.Errorf(
			"%s: owner lifetime burn limit exceeded: historical=%d candidate_prior=%d current=%d total=%d max=%d atoms",
			BurnOwnerLifetimePolicyVersionV1,
			historicalBurnAtoms,
			candidatePriorBurnAtoms,
			classification.BurnAmountAtoms,
			total,
			maxAtoms,
		)
	}

	return nil
}
