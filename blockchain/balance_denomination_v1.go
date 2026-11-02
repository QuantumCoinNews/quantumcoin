package blockchain

import (
	"fmt"

	"quantumcoin/wallet"
)

// balanceInProtocolUnitsAtHeightLockedV1 calculates a wallet balance in the
// integer monetary unit required by spendHeight.
//
// The caller must hold bc.stateMu for the complete height/origin/UTXO snapshot.
// Historical outputs themselves are never rewritten.
func (bc *Blockchain) balanceInProtocolUnitsAtHeightLockedV1(
	address string,
	spendHeight int,
	spendableOnly bool,
) int {
	if bc == nil || spendHeight < 0 {
		return 0
	}

	pubKeyHash := wallet.Base58DecodeAddress(address)
	if len(pubKeyHash) == 0 {
		return 0
	}

	origins := bc.utxoOriginsV1()
	total := 0

	for txID, outputs := range bc.UTXO {
		for outputIndex, output := range outputs {
			// UpdateUTXOSet preserves spent output positions as zero-value
			// slots. They must never contribute to a balance.
			if output.Amount <= 0 {
				continue
			}

			if !output.IsLockedWithKey(pubKeyHash) {
				continue
			}

			outpoint := fmt.Sprintf(
				"%s:%d",
				txID,
				outputIndex,
			)

			origin, ok := origins[outpoint]
			if !ok {
				// Fail closed rather than mixing an amount whose historical
				// denomination cannot be established.
				return 0
			}

			if spendableOnly {
				if err := bc.validateCoinbaseMaturityAtHeightV1(
					origin,
					outpoint,
					spendHeight,
				); err != nil {
					continue
				}
			}

			normalized, err :=
				bc.normalizeUTXOAmountForSpendV1(
					output.Amount,
					origin.Height,
					spendHeight,
				)
			if err != nil {
				return 0
			}

			total, err = addPositiveUTXOAmountV1(
				total,
				normalized,
				"wallet balance",
			)
			if err != nil {
				return 0
			}
		}
	}

	return total
}
