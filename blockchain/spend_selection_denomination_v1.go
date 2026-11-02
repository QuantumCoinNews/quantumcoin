package blockchain

import (
	"encoding/hex"
	"fmt"
	"math"
)

const SpendSelectionDenominationVersionV1 = "QC_SPEND_SELECTION_DENOMINATION_V1"

// FindSpendableOutputsAtHeightV1 selects UTXOs in the monetary unit required
// by spendHeight. Historical outputs remain stored in their original unit.
func (bc *Blockchain) FindSpendableOutputsAtHeightV1(
	pubKeyHash []byte,
	amount int,
	spendHeight int,
) (map[string][]int, int, error) {
	unspent := make(map[string][]int)
	accumulated := 0

	if bc == nil {
		return unspent, 0, fmt.Errorf(
			"%s: nil blockchain",
			SpendSelectionDenominationVersionV1,
		)
	}

	if amount <= 0 {
		return unspent, 0, fmt.Errorf(
			"%s: amount must be positive",
			SpendSelectionDenominationVersionV1,
		)
	}

	if len(pubKeyHash) == 0 {
		return unspent, 0, fmt.Errorf(
			"%s: empty public key hash",
			SpendSelectionDenominationVersionV1,
		)
	}

	if spendHeight < 0 {
		return unspent, 0, fmt.Errorf(
			"%s: invalid spend height %d",
			SpendSelectionDenominationVersionV1,
			spendHeight,
		)
	}

	if bc.UTXO == nil {
		bc.UpdateUTXOSet()
	}

	origins := bc.utxoOriginsV1()

	for txID, outputs := range bc.UTXO {
		rawTxID, err := hex.DecodeString(txID)
		if err != nil {
			continue
		}

		for outputIndex, output := range outputs {
			if output.Amount <= 0 {
				continue
			}

			if !output.IsLockedWithKey(pubKeyHash) {
				continue
			}

			outpoint := outpointKeyV1(
				rawTxID,
				outputIndex,
			)

			origin, exists := origins[outpoint]
			if !exists {
				return nil, 0, fmt.Errorf(
					"%s: missing UTXO origin: %s",
					SpendSelectionDenominationVersionV1,
					outpoint,
				)
			}

			if err := bc.validateCoinbaseMaturityAtHeightV1(
				origin,
				outpoint,
				spendHeight,
			); err != nil {
				continue
			}

			normalized, err :=
				bc.normalizeUTXOAmountForSpendV1(
					output.Amount,
					origin.Height,
					spendHeight,
				)
			if err != nil {
				return nil, 0, fmt.Errorf(
					"%s: normalize %s: %w",
					SpendSelectionDenominationVersionV1,
					outpoint,
					err,
				)
			}

			if normalized <= 0 {
				return nil, 0, fmt.Errorf(
					"%s: normalized amount must be positive",
					SpendSelectionDenominationVersionV1,
				)
			}

			if accumulated > math.MaxInt-normalized {
				return nil, 0, fmt.Errorf(
					"%s: accumulated amount overflow",
					SpendSelectionDenominationVersionV1,
				)
			}

			accumulated += normalized
			unspent[txID] = append(
				unspent[txID],
				outputIndex,
			)

			if accumulated >= amount {
				return unspent, accumulated, nil
			}
		}
	}

	return unspent, accumulated, nil
}
