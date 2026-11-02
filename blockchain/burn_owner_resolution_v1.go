package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

const BurnOwnerResolutionVersionV1 = "QC_BURN_OWNER_RESOLUTION_V1"

// ResolveCanonicalBurnOwnerKeyV1 derives the accounting owner of a canonical
// burn transaction from the current canonical UTXO state.
//
// tx.Sender is intentionally not used for attribution.
func (bc *Blockchain) ResolveCanonicalBurnOwnerKeyV1(
	tx *Transaction,
) ([]byte, error) {
	if bc == nil {
		return nil, fmt.Errorf(
			"%s: nil blockchain",
			BurnOwnerResolutionVersionV1,
		)
	}

	if bc.UTXO == nil {
		bc.UpdateUTXOSet()
	}

	return bc.resolveCanonicalBurnOwnerKeyAgainstUTXOStateAtHeightV1(
		tx,
		bc.UTXO,
		bc.nextSpendHeightV1(),
	)
}

// resolveCanonicalBurnOwnerKeyAgainstUTXOStateV1 preserves the current-height
// helper contract while delegating to the explicit-height implementation.
func (bc *Blockchain) resolveCanonicalBurnOwnerKeyAgainstUTXOStateV1(
	tx *Transaction,
	utxoState map[string][]TransactionOutput,
) ([]byte, error) {
	if bc == nil {
		return nil, fmt.Errorf(
			"%s: nil blockchain",
			BurnOwnerResolutionVersionV1,
		)
	}

	return bc.resolveCanonicalBurnOwnerKeyAgainstUTXOStateAtHeightV1(
		tx,
		utxoState,
		bc.nextSpendHeightV1(),
	)
}

// resolveCanonicalBurnOwnerKeyAgainstUTXOStateAtHeightV1 derives the
// accounting owner from the exact candidate UTXO state and spend height
// supplied by the caller.
//
// tx.Sender is intentionally not used. Every burn input must unlock an
// existing non-burn UTXO with the same public-key hash. Mixed-owner burns are
// rejected so lifetime burn accounting has one deterministic owner key.
func (bc *Blockchain) resolveCanonicalBurnOwnerKeyAgainstUTXOStateAtHeightV1(
	tx *Transaction,
	utxoState map[string][]TransactionOutput,
	spendHeight int,
) ([]byte, error) {
	if bc == nil {
		return nil, fmt.Errorf(
			"%s: nil blockchain",
			BurnOwnerResolutionVersionV1,
		)
	}

	if tx == nil {
		return nil, fmt.Errorf(
			"%s: nil transaction",
			BurnOwnerResolutionVersionV1,
		)
	}

	if utxoState == nil {
		return nil, fmt.Errorf(
			"%s: nil UTXO state",
			BurnOwnerResolutionVersionV1,
		)
	}

	if spendHeight < 0 {
		return nil, fmt.Errorf(
			"%s: invalid spend height %d",
			BurnOwnerResolutionVersionV1,
			spendHeight,
		)
	}

	classification, err :=
		ClassifyCanonicalBurnTransactionAtHeightV1(
			tx,
			spendHeight,
			bc,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: classify burn transaction: %w",
			BurnOwnerResolutionVersionV1,
			err,
		)
	}

	if !classification.IsBurn {
		return nil, fmt.Errorf(
			"%s: transaction is not a canonical burn",
			BurnOwnerResolutionVersionV1,
		)
	}

	if len(tx.Inputs) == 0 {
		return nil, fmt.Errorf(
			"%s: burn transaction has no inputs",
			BurnOwnerResolutionVersionV1,
		)
	}

	var ownerKey []byte

	for inputIndex, input := range tx.Inputs {
		if len(input.TxID) == 0 ||
			input.OutIndex < 0 {
			return nil, fmt.Errorf(
				"%s: input[%d] invalid outpoint",
				BurnOwnerResolutionVersionV1,
				inputIndex,
			)
		}

		txIDHex :=
			hex.EncodeToString(
				input.TxID,
			)

		outputs, ok := utxoState[txIDHex]
		if !ok ||
			input.OutIndex >= len(outputs) {
			return nil, fmt.Errorf(
				"%s: input[%d] missing utxo",
				BurnOwnerResolutionVersionV1,
				inputIndex,
			)
		}

		previousOutput :=
			outputs[input.OutIndex]

		if IsCanonicalBurnOutputV1(
			previousOutput,
		) {
			return nil, fmt.Errorf(
				"%s: input[%d] references canonical burn output",
				BurnOwnerResolutionVersionV1,
				inputIndex,
			)
		}

		if previousOutput.Amount <= 0 {
			return nil, fmt.Errorf(
				"%s: input[%d] references spent or empty utxo",
				BurnOwnerResolutionVersionV1,
				inputIndex,
			)
		}

		pubKeyHash, err :=
			inputPubKeyHashV1(
				input.PubKey,
			)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: input[%d] owner key: %w",
				BurnOwnerResolutionVersionV1,
				inputIndex,
				err,
			)
		}

		if !previousOutput.IsLockedWithKey(
			pubKeyHash,
		) {
			return nil, fmt.Errorf(
				"%s: input[%d] public key does not own utxo",
				BurnOwnerResolutionVersionV1,
				inputIndex,
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
				"%s: mixed-owner burn transaction",
				BurnOwnerResolutionVersionV1,
			)
		}
	}

	if len(ownerKey) == 0 {
		return nil, fmt.Errorf(
			"%s: burn owner could not be resolved",
			BurnOwnerResolutionVersionV1,
		)
	}

	return ownerKey, nil
}
