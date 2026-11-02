package blockchain

import (
	"fmt"
	"math"

	qconsensus "quantumcoin/consensus"
)

const BurnTransactionClassificationVersionV1 = "QC_BURN_TRANSACTION_CLASSIFICATION_V1"

type BurnTransactionClassificationV1 struct {
	IsBurn          bool
	BurnAmountAtoms int64
}

func ClassifyCanonicalBurnTransactionAtHeightV1(
	tx *Transaction,
	spendHeight int,
	bc *Blockchain,
) (BurnTransactionClassificationV1, error) {
	var result BurnTransactionClassificationV1

	if bc == nil {
		return result, fmt.Errorf(
			"%s: nil blockchain",
			BurnTransactionClassificationVersionV1,
		)
	}

	if tx == nil {
		return result, fmt.Errorf(
			"%s: nil transaction",
			BurnTransactionClassificationVersionV1,
		)
	}

	if spendHeight < 0 {
		return result, fmt.Errorf(
			"%s: invalid spend height %d",
			BurnTransactionClassificationVersionV1,
			spendHeight,
		)
	}

	protocol := bc.ProtocolAtHeightV1(
		spendHeight,
	)

	for outputIndex, output := range tx.Outputs {
		if !IsCanonicalBurnOutputV1(output) {
			continue
		}

		rawAmount := int64(output.Amount)
		if rawAmount <= 0 {
			return result, fmt.Errorf(
				"%s: non-positive burn output at index %d",
				BurnTransactionClassificationVersionV1,
				outputIndex,
			)
		}

		var atoms int64

		if qconsensus.ProtocolUsesAtomDenominationV1(
			protocol,
		) {
			atoms = rawAmount
		} else {
			converted, err :=
				qconsensus.WholeQCToAtomsV1(
					rawAmount,
				)
			if err != nil {
				return result, fmt.Errorf(
					"%s: burn output %d conversion: %w",
					BurnTransactionClassificationVersionV1,
					outputIndex,
					err,
				)
			}

			atoms = converted
		}

		if atoms <= 0 {
			return result, fmt.Errorf(
				"%s: non-positive burn atoms at output %d",
				BurnTransactionClassificationVersionV1,
				outputIndex,
			)
		}

		if result.BurnAmountAtoms >
			math.MaxInt64-atoms {
			return result, fmt.Errorf(
				"%s: burn amount overflow",
				BurnTransactionClassificationVersionV1,
			)
		}

		result.BurnAmountAtoms += atoms
		result.IsBurn = true
	}

	return result, nil
}
