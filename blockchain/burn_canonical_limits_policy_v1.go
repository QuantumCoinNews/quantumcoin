package blockchain

import (
	"fmt"

	qconsensus "quantumcoin/consensus"
)

const BurnCanonicalLimitsPolicyVersionV1 = "QC_BURN_CANONICAL_LIMITS_POLICY_V1"

func (bc *Blockchain) validateCanonicalBurnLimitsAtHeightV1(
	tx *Transaction,
	spendHeight int,
	ownerKey []byte,
	candidatePriorOwnerBurnAtoms int64,
	candidatePriorNetworkBurnAtoms int64,
) error {
	if bc == nil {
		return fmt.Errorf(
			"%s: nil blockchain",
			BurnCanonicalLimitsPolicyVersionV1,
		)
	}

	if tx == nil {
		return fmt.Errorf(
			"%s: nil transaction",
			BurnCanonicalLimitsPolicyVersionV1,
		)
	}

	if spendHeight < 0 {
		return fmt.Errorf(
			"%s: invalid spend height: %d",
			BurnCanonicalLimitsPolicyVersionV1,
			spendHeight,
		)
	}

	if err := bc.validateCanonicalBurnPerTransactionLimitAtHeightV1(
		tx,
		spendHeight,
		qconsensus.CanonicalBurnPerTransactionMaxAtomsV1,
	); err != nil {
		return fmt.Errorf(
			"%s: per-transaction limit: %w",
			BurnCanonicalLimitsPolicyVersionV1,
			err,
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
			BurnCanonicalLimitsPolicyVersionV1,
			err,
		)
	}

	if !classification.IsBurn {
		return nil
	}

	if err := bc.validateCanonicalBurnOwnerLifetimeLimitAtHeightV1(
		tx,
		spendHeight,
		ownerKey,
		candidatePriorOwnerBurnAtoms,
		qconsensus.CanonicalBurnOwnerLifetimeMaxAtomsV1,
	); err != nil {
		return fmt.Errorf(
			"%s: owner lifetime limit: %w",
			BurnCanonicalLimitsPolicyVersionV1,
			err,
		)
	}

	if err := bc.validateCanonicalBurnNetworkLifetimeLimitAtHeightV1(
		tx,
		spendHeight,
		candidatePriorNetworkBurnAtoms,
		qconsensus.CanonicalBurnNetworkLifetimeMaxAtomsV1,
	); err != nil {
		return fmt.Errorf(
			"%s: network lifetime limit: %w",
			BurnCanonicalLimitsPolicyVersionV1,
			err,
		)
	}

	return nil
}
