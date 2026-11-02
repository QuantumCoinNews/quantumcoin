package blockchain

import (
	"bytes"
	"fmt"
	"math"
)

const BurnOwnerLifetimeAccountingVersionV1 = "QC_BURN_OWNER_LIFETIME_ACCOUNTING_V1"

// TotalBurnedAtomsByOwnerCheckedV1 derives the lifetime amount burned by one
// owner from the current canonical chain.
//
// Historical ownership attribution relies only on canonical transactions that
// have already passed consensus validation. tx.Sender is not used.
//
// The result is always returned in atoms, with denomination selected from the
// canonical protocol active at each block height.
func (bc *Blockchain) TotalBurnedAtomsByOwnerCheckedV1(
	ownerKey []byte,
) (int64, error) {
	if bc == nil {
		return 0, fmt.Errorf(
			"%s: nil blockchain",
			BurnOwnerLifetimeAccountingVersionV1,
		)
	}

	if len(ownerKey) == 0 {
		return 0, fmt.Errorf(
			"%s: empty owner key",
			BurnOwnerLifetimeAccountingVersionV1,
		)
	}

	var total int64

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}

		if block.Index < 0 {
			return 0, fmt.Errorf(
				"%s: invalid block height %d",
				BurnOwnerLifetimeAccountingVersionV1,
				block.Index,
			)
		}

		for txIndex, tx := range block.Transactions {
			if tx == nil {
				continue
			}

			classification, err :=
				ClassifyCanonicalBurnTransactionAtHeightV1(
					tx,
					block.Index,
					bc,
				)
			if err != nil {
				return 0, fmt.Errorf(
					"%s: classify tx[%d] at height %d: %w",
					BurnOwnerLifetimeAccountingVersionV1,
					txIndex,
					block.Index,
					err,
				)
			}

			if !classification.IsBurn {
				continue
			}

			historicalOwner, err :=
				ResolveValidatedHistoricalBurnOwnerKeyV1(
					tx,
				)
			if err != nil {
				return 0, fmt.Errorf(
					"%s: resolve burn owner tx[%d] at height %d: %w",
					BurnOwnerLifetimeAccountingVersionV1,
					txIndex,
					block.Index,
					err,
				)
			}

			if !bytes.Equal(
				historicalOwner,
				ownerKey,
			) {
				continue
			}

			if classification.BurnAmountAtoms <= 0 {
				return 0, fmt.Errorf(
					"%s: invalid burn amount tx[%d] at height %d: %d atoms",
					BurnOwnerLifetimeAccountingVersionV1,
					txIndex,
					block.Index,
					classification.BurnAmountAtoms,
				)
			}

			if total >
				math.MaxInt64-classification.BurnAmountAtoms {
				return 0, fmt.Errorf(
					"%s: owner lifetime burn overflow at height %d",
					BurnOwnerLifetimeAccountingVersionV1,
					block.Index,
				)
			}

			total +=
				classification.BurnAmountAtoms
		}
	}

	return total, nil
}
