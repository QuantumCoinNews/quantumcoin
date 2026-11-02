package blockchain

import (
	"fmt"
)

const BurnConsensusPolicyVersionV1 = "QC_BURN_CONSENSUS_POLICY_V1"

// validateCanonicalBurnPerTransactionLimitAtHeightV1 applies only the
// per-transaction boundary of the burn consensus policy.
//
// maxAtoms is supplied explicitly so the economic limit can be fixed in the
// canonical economy policy separately. This helper must not invent or embed
// that policy value.
//
// Non-burn transactions are unaffected.
func (bc *Blockchain) validateCanonicalBurnPerTransactionLimitAtHeightV1(
	tx *Transaction,
	spendHeight int,
	maxAtoms int64,
) error {
	if bc == nil {
		return fmt.Errorf(
			"%s: nil blockchain",
			BurnConsensusPolicyVersionV1,
		)
	}

	if tx == nil {
		return fmt.Errorf(
			"%s: nil transaction",
			BurnConsensusPolicyVersionV1,
		)
	}

	if spendHeight < 0 {
		return fmt.Errorf(
			"%s: invalid spend height %d",
			BurnConsensusPolicyVersionV1,
			spendHeight,
		)
	}

	if maxAtoms <= 0 {
		return fmt.Errorf(
			"%s: invalid per-transaction burn limit: %d atoms",
			BurnConsensusPolicyVersionV1,
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
			BurnConsensusPolicyVersionV1,
			err,
		)
	}

	if !classification.IsBurn {
		return nil
	}

	if classification.BurnAmountAtoms <= 0 {
		return fmt.Errorf(
			"%s: invalid canonical burn amount: %d atoms",
			BurnConsensusPolicyVersionV1,
			classification.BurnAmountAtoms,
		)
	}

	if classification.BurnAmountAtoms > maxAtoms {
		return fmt.Errorf(
			"%s: per-transaction burn limit exceeded: burn=%d max=%d atoms",
			BurnConsensusPolicyVersionV1,
			classification.BurnAmountAtoms,
			maxAtoms,
		)
	}

	return nil
}
