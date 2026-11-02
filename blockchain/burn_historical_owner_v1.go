package blockchain

import (
	"bytes"
	"fmt"
)

const BurnHistoricalOwnerVersionV1 = "QC_BURN_HISTORICAL_OWNER_V1"

// ResolveValidatedHistoricalBurnOwnerKeyV1 derives the accounting owner from
// the input public keys of a burn transaction that has already passed
// canonical consensus validation.
//
// This helper does not independently prove historical UTXO ownership.
// That proof belongs to canonical UTXO validation. It is intended only for
// deterministic accounting over already-validated canonical chain history.
func ResolveValidatedHistoricalBurnOwnerKeyV1(
	tx *Transaction,
) ([]byte, error) {
	if tx == nil {
		return nil, fmt.Errorf(
			"%s: nil transaction",
			BurnHistoricalOwnerVersionV1,
		)
	}

	if len(tx.Inputs) == 0 {
		return nil, fmt.Errorf(
			"%s: transaction has no inputs",
			BurnHistoricalOwnerVersionV1,
		)
	}

	var ownerKey []byte

	for inputIndex, input := range tx.Inputs {
		pubKeyHash, err :=
			inputPubKeyHashV1(
				input.PubKey,
			)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: input[%d] owner key: %w",
				BurnHistoricalOwnerVersionV1,
				inputIndex,
				err,
			)
		}

		if len(ownerKey) == 0 {
			ownerKey =
				append(
					[]byte(nil),
					pubKeyHash...,
				)
			continue
		}

		if !bytes.Equal(
			ownerKey,
			pubKeyHash,
		) {
			return nil, fmt.Errorf(
				"%s: mixed-owner historical burn transaction",
				BurnHistoricalOwnerVersionV1,
			)
		}
	}

	if len(ownerKey) == 0 {
		return nil, fmt.Errorf(
			"%s: historical burn owner could not be resolved",
			BurnHistoricalOwnerVersionV1,
		)
	}

	return ownerKey, nil
}
