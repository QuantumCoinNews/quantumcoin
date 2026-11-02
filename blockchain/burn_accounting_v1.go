package blockchain

import (
	"fmt"
	"math"

	qconsensus "quantumcoin/consensus"
)

const BurnAccountingVersionV1 = "QC_BURN_ACCOUNTING_V1"

// TotalBurnedAtomsCheckedV1 derives the total amount permanently burned by
// scanning the current canonical chain.
//
// No mutable burn counter is stored. Therefore a canonical-chain replacement
// or reorg is reflected automatically by the current bc.Blocks view.
//
// Historical output denomination is determined by the output's containing
// block height:
//   - LegacyV1 / ContinuityV2 amounts are whole-QC integers.
//   - AtomV3 amounts are atom integers.
//
// Accounting failures and integer overflow fail closed.
//
// TotalSupply is the configured whole-QC hard cap and is never modified here.
func (bc *Blockchain) TotalBurnedAtomsCheckedV1() (int64, error) {
	if bc == nil {
		return 0, fmt.Errorf(
			"%s: nil blockchain",
			BurnAccountingVersionV1,
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
				BurnAccountingVersionV1,
				block.Index,
			)
		}

		protocol := bc.ProtocolAtHeightV1(block.Index)

		for _, tx := range block.Transactions {
			if tx == nil {
				continue
			}

			for _, output := range tx.Outputs {
				if !IsCanonicalBurnOutputV1(output) {
					continue
				}

				raw := int64(output.Amount)
				var atoms int64

				if qconsensus.ProtocolUsesAtomDenominationV1(protocol) {
					atoms = raw
				} else {
					converted, err :=
						qconsensus.WholeQCToAtomsV1(raw)
					if err != nil {
						return 0, fmt.Errorf(
							"%s: burn amount conversion at height %d: %w",
							BurnAccountingVersionV1,
							block.Index,
							err,
						)
					}
					atoms = converted
				}

				if atoms <= 0 {
					return 0, fmt.Errorf(
						"%s: non-positive canonical burn amount at height %d",
						BurnAccountingVersionV1,
						block.Index,
					)
				}

				if total > math.MaxInt64-atoms {
					return 0, fmt.Errorf(
						"%s: total burned atom overflow at height %d",
						BurnAccountingVersionV1,
						block.Index,
					)
				}

				total += atoms
			}
		}
	}

	return total, nil
}

// TotalBurnedAtomsV1 is the compatibility/read-only view.
//
// Consensus/economy decisions that must distinguish accounting failure from
// a genuine zero-burn chain must use TotalBurnedAtomsCheckedV1.
func (bc *Blockchain) TotalBurnedAtomsV1() int64 {
	total, err := bc.TotalBurnedAtomsCheckedV1()
	if err != nil {
		return 0
	}

	return total
}
