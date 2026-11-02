package blockchain

import (
	"encoding/hex"
	"fmt"
	"time"

	"quantumcoin/wallet"
)

const BurnTransactionBuilderVersionV1 = "QC_BURN_TRANSACTION_BUILDER_V1"

// NewCanonicalBurnTransactionFromQCAmountV1 builds an explicit user burn
// transaction from an exact user-facing QC amount.
//
// The transaction spends only UTXOs owned by from. The burned amount is sent
// to the canonical unspendable burn output marker. Any excess input value is
// returned to from as a normal change output.
//
// Signing remains the caller's responsibility and uses Transaction.Sign.
func NewCanonicalBurnTransactionFromQCAmountV1(
	from string,
	rawQC string,
	bc *Blockchain,
) (*Transaction, error) {
	if bc == nil {
		return nil, fmt.Errorf(
			"%s: nil blockchain",
			BurnTransactionBuilderVersionV1,
		)
	}

	bc.stateMu.Lock()
	defer bc.stateMu.Unlock()

	spendHeight := bc.nextSpendHeightV1()

	amountUnits, err :=
		bc.qcAmountStringToTransactionUnitsAtHeightV1(
			rawQC,
			spendHeight,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: convert exact QC burn amount: %w",
			BurnTransactionBuilderVersionV1,
			err,
		)
	}

	if amountUnits <= 0 {
		return nil, fmt.Errorf(
			"%s: invalid burn amount",
			BurnTransactionBuilderVersionV1,
		)
	}

	pubFrom := wallet.Base58DecodeAddress(from)
	if len(pubFrom) == 0 {
		return nil, fmt.Errorf(
			"%s: invalid sender address",
			BurnTransactionBuilderVersionV1,
		)
	}

	utxos, acc, err :=
		bc.FindSpendableOutputsAtHeightV1(
			pubFrom,
			amountUnits,
			spendHeight,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: find spendable outputs: %w",
			BurnTransactionBuilderVersionV1,
			err,
		)
	}

	if acc < amountUnits {
		return nil, fmt.Errorf(
			"%s: insufficient balance",
			BurnTransactionBuilderVersionV1,
		)
	}

	var inputs []TransactionInput

	for txid, outs := range utxos {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: decode spendable txid: %w",
				BurnTransactionBuilderVersionV1,
				err,
			)
		}

		for _, outIdx := range outs {
			inputs = append(
				inputs,
				TransactionInput{
					TxID:     txID,
					OutIndex: outIdx,
				},
			)
		}
	}

	if len(inputs) == 0 {
		return nil, fmt.Errorf(
			"%s: no spendable inputs",
			BurnTransactionBuilderVersionV1,
		)
	}

	outputs := []TransactionOutput{
		NewCanonicalBurnOutputV1(amountUnits),
	}

	if acc > amountUnits {
		outputs = append(
			outputs,
			TransactionOutput{
				Amount:     acc - amountUnits,
				PubKeyHash: append([]byte(nil), pubFrom...),
			},
		)
	}

	tx := &Transaction{
		ID:        nil,
		Inputs:    inputs,
		Outputs:   outputs,
		Timestamp: time.Now(),
		Sender:    from,
		Amount:    float64(amountUnits),
	}

	tx.ID = tx.Hash()

	return tx, nil
}
