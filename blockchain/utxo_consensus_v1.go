package blockchain

import (
	"encoding/hex"
	"fmt"

	"quantumcoin/wallet"
)

const UTXOConsensusVersionV1 = "QC_CANONICAL_UTXO_VALIDATION_V1"

type UTXOValidationResultV1 struct {
	InputTotal  int
	OutputTotal int
	Fee         int
}

// UTXOOriginV1 records the consensus-relevant provenance of an outpoint.
// Coinbase maturity is evaluated against the candidate spend height, never
// against the node's current wall-clock time or an implicit chain position.
type UTXOOriginV1 struct {
	Height   int
	Coinbase bool
}

func outpointKeyV1(txID []byte, outIndex int) string {
	return fmt.Sprintf("%x:%d", txID, outIndex)
}

func (bc *Blockchain) nextSpendHeightV1() int {
	if bc == nil || len(bc.Blocks) == 0 {
		return 0
	}

	last := bc.Blocks[len(bc.Blocks)-1]
	if last == nil {
		return len(bc.Blocks)
	}

	return last.Index + 1
}

func (bc *Blockchain) utxoOriginsV1() map[string]UTXOOriginV1 {
	origins := make(map[string]UTXOOriginV1)
	if bc == nil {
		return origins
	}

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}
		for _, tx := range block.Transactions {
			if tx == nil || len(tx.ID) == 0 {
				continue
			}
			for outputIndex := range tx.Outputs {
				origins[outpointKeyV1(tx.ID, outputIndex)] = UTXOOriginV1{
					Height:   block.Index,
					Coinbase: tx.IsCoinbase(),
				}
			}
		}
	}

	return origins
}

func (bc *Blockchain) validateCoinbaseMaturityAtHeightV1(
	origin UTXOOriginV1,
	outpoint string,
	spendHeight int,
) error {
	if !origin.Coinbase || bc == nil || bc.coinbaseMaturity <= 0 {
		return nil
	}
	if spendHeight < 0 {
		return fmt.Errorf(
			"%s: invalid candidate spend height %d",
			UTXOConsensusVersionV1,
			spendHeight,
		)
	}

	age := spendHeight - origin.Height
	if age < bc.coinbaseMaturity {
		return fmt.Errorf(
			"%s: coinbase output is not mature: %s origin_height=%d spend_height=%d age=%d maturity=%d",
			UTXOConsensusVersionV1,
			outpoint,
			origin.Height,
			spendHeight,
			age,
			bc.coinbaseMaturity,
		)
	}

	return nil
}

func cloneUTXOSetV1(
	source map[string][]TransactionOutput,
) map[string][]TransactionOutput {
	cloned := make(map[string][]TransactionOutput, len(source))
	for txID, outputs := range source {
		cloned[txID] = append([]TransactionOutput(nil), outputs...)
	}
	return cloned
}

func addPositiveUTXOAmountV1(
	total int,
	amount int,
	label string,
) (int, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("%s: %s must be positive: %d", UTXOConsensusVersionV1, label, amount)
	}

	next := total + amount
	if next < total {
		return 0, fmt.Errorf("%s: %s overflow", UTXOConsensusVersionV1, label)
	}

	return next, nil
}

func inputPubKeyHashV1(pubKey []byte) ([]byte, error) {
	if len(pubKey) == 0 {
		return nil, fmt.Errorf("%s: missing input pubkey", UTXOConsensusVersionV1)
	}

	normalized := pubKey
	if len(normalized) == 65 && normalized[0] == 0x04 {
		normalized = normalized[1:]
	}
	if len(normalized) != 64 {
		return nil, fmt.Errorf(
			"%s: invalid input pubkey length %d",
			UTXOConsensusVersionV1,
			len(pubKey),
		)
	}

	return wallet.HashPubKey(normalized), nil
}

func (bc *Blockchain) knownTransactionIDsV1() map[string]bool {
	known := make(map[string]bool)
	if bc == nil {
		return known
	}

	for _, block := range bc.Blocks {
		if block == nil {
			continue
		}
		for _, tx := range block.Transactions {
			if tx == nil || len(tx.ID) == 0 {
				continue
			}
			known[hex.EncodeToString(tx.ID)] = true
		}
	}

	return known
}

func (bc *Blockchain) validateTransactionAgainstUTXOStateV1(
	tx *Transaction,
	scratch map[string][]TransactionOutput,
	origins map[string]UTXOOriginV1,
	createdNormalTx map[string]bool,
	spentInSequence map[string]bool,
	knownTxIDs map[string]bool,
	checkPending bool,
	spendHeight int,
) (UTXOValidationResultV1, error) {
	var result UTXOValidationResultV1

	if bc == nil {
		return result, fmt.Errorf("%s: nil blockchain", UTXOConsensusVersionV1)
	}
	if tx == nil {
		return result, ErrNilTransaction
	}
	if tx.IsCoinbase() {
		return result, fmt.Errorf("%s: coinbase is not a normal UTXO spend", UTXOConsensusVersionV1)
	}
	if scratch == nil {
		return result, fmt.Errorf("%s: nil UTXO state", UTXOConsensusVersionV1)
	}
	if origins == nil || createdNormalTx == nil || spentInSequence == nil || knownTxIDs == nil {
		return result, fmt.Errorf("%s: incomplete UTXO validation state", UTXOConsensusVersionV1)
	}
	if spendHeight < 0 {
		return result, fmt.Errorf(
			"%s: invalid candidate spend height %d",
			UTXOConsensusVersionV1,
			spendHeight,
		)
	}

	if err := ValidateTransactionBasic(tx); err != nil {
		return result, fmt.Errorf(
			"%s: basic validation: %w",
			UTXOConsensusVersionV1,
			err,
		)
	}
	if !tx.Verify() {
		return result, fmt.Errorf(
			"%s: invalid transaction signature",
			UTXOConsensusVersionV1,
		)
	}
	if len(tx.Inputs) == 0 {
		return result, fmt.Errorf(
			"%s: transaction has no inputs",
			UTXOConsensusVersionV1,
		)
	}

	txIDHex := hex.EncodeToString(tx.ID)
	if txIDHex == "" {
		return result, fmt.Errorf(
			"%s: missing transaction id",
			UTXOConsensusVersionV1,
		)
	}
	if knownTxIDs[txIDHex] || createdNormalTx[txIDHex] {
		return result, fmt.Errorf(
			"%s: duplicate transaction id: %s",
			UTXOConsensusVersionV1,
			txIDHex,
		)
	}

	for outputIndex, output := range tx.Outputs {
		var err error

		result.OutputTotal, err = addPositiveUTXOAmountV1(
			result.OutputTotal,
			output.Amount,
			fmt.Sprintf("output[%d] amount", outputIndex),
		)
		if err != nil {
			return UTXOValidationResultV1{}, err
		}
	}

	for inputIndex, input := range tx.Inputs {
		if len(input.TxID) == 0 || input.OutIndex < 0 {
			return UTXOValidationResultV1{}, fmt.Errorf(
				"%s: input[%d] invalid outpoint",
				UTXOConsensusVersionV1,
				inputIndex,
			)
		}

		outpoint := outpointKeyV1(
			input.TxID,
			input.OutIndex,
		)

		if spentInSequence[outpoint] {
			return UTXOValidationResultV1{}, fmt.Errorf(
				"%s: double spend in validation sequence: %s",
				UTXOConsensusVersionV1,
				outpoint,
			)
		}

		if checkPending &&
			bc.pendingSpendsOutpoint(
				input.TxID,
				input.OutIndex,
			) {
			return UTXOValidationResultV1{}, fmt.Errorf(
				"%s: input already pending in mempool: %s",
				UTXOConsensusVersionV1,
				outpoint,
			)
		}

		inputTxIDHex := hex.EncodeToString(input.TxID)

		outputs, ok := scratch[inputTxIDHex]
		if !ok || input.OutIndex >= len(outputs) {
			return UTXOValidationResultV1{}, fmt.Errorf(
				"%s: missing utxo: %s",
				UTXOConsensusVersionV1,
				outpoint,
			)
		}

		previousOutput := outputs[input.OutIndex]

		if IsCanonicalBurnOutputV1(previousOutput) {
			return UTXOValidationResultV1{}, fmt.Errorf(
				"%s: canonical burn output is unspendable: %s",
				UTXOConsensusVersionV1,
				outpoint,
			)
		}
		if previousOutput.Amount <= 0 {
			return UTXOValidationResultV1{}, fmt.Errorf(
				"%s: spent or empty utxo: %s",
				UTXOConsensusVersionV1,
				outpoint,
			)
		}

		normalizedInputAmount := previousOutput.Amount

		if !createdNormalTx[inputTxIDHex] {
			origin, exists := origins[outpoint]
			if !exists {
				return UTXOValidationResultV1{}, fmt.Errorf(
					"%s: missing utxo origin: %s",
					UTXOConsensusVersionV1,
					outpoint,
				)
			}

			if err := bc.validateCoinbaseMaturityAtHeightV1(
				origin,
				outpoint,
				spendHeight,
			); err != nil {
				return UTXOValidationResultV1{}, err
			}

			var err error
			normalizedInputAmount, err =
				bc.normalizeUTXOAmountForSpendV1(
					previousOutput.Amount,
					origin.Height,
					spendHeight,
				)
			if err != nil {
				return UTXOValidationResultV1{}, fmt.Errorf(
					"%s: normalize input[%d] amount: %w",
					UTXOConsensusVersionV1,
					inputIndex,
					err,
				)
			}
		}

		pubKeyHash, err :=
			inputPubKeyHashV1(input.PubKey)
		if err != nil {
			return UTXOValidationResultV1{}, fmt.Errorf(
				"%s: input[%d] %w",
				UTXOConsensusVersionV1,
				inputIndex,
				err,
			)
		}

		if !previousOutput.IsLockedWithKey(pubKeyHash) {
			return UTXOValidationResultV1{}, fmt.Errorf(
				"%s: input pubkey does not unlock utxo: %s",
				UTXOConsensusVersionV1,
				outpoint,
			)
		}

		result.InputTotal, err =
			addPositiveUTXOAmountV1(
				result.InputTotal,
				normalizedInputAmount,
				fmt.Sprintf(
					"input[%d] amount",
					inputIndex,
				),
			)
		if err != nil {
			return UTXOValidationResultV1{}, err
		}

		spentInSequence[outpoint] = true
		outputs[input.OutIndex] =
			TransactionOutput{}
		scratch[inputTxIDHex] = outputs
	}

	if result.InputTotal < result.OutputTotal {
		return UTXOValidationResultV1{}, fmt.Errorf(
			"%s: insufficient input amount: input=%d output=%d",
			UTXOConsensusVersionV1,
			result.InputTotal,
			result.OutputTotal,
		)
	}

	result.Fee =
		result.InputTotal - result.OutputTotal

	scratch[txIDHex] =
		append(
			[]TransactionOutput(nil),
			tx.Outputs...,
		)

	createdNormalTx[txIDHex] = true
	knownTxIDs[txIDHex] = true

	return result, nil
}
func (bc *Blockchain) ValidateTransactionUTXOAtHeightV1(
	tx *Transaction,
	checkPending bool,
	spendHeight int,
) (UTXOValidationResultV1, error) {
	if bc == nil {
		return UTXOValidationResultV1{}, fmt.Errorf("%s: nil blockchain", UTXOConsensusVersionV1)
	}
	if bc.UTXO == nil {
		bc.UpdateUTXOSet()
	}

	known := bc.knownTransactionIDsV1()
	if checkPending {
		for _, pending := range bc.pendingTxs {
			if pending == nil || len(pending.ID) == 0 {
				continue
			}
			known[hex.EncodeToString(pending.ID)] = true
		}
	}

	return bc.validateTransactionAgainstUTXOStateV1(
		tx,
		cloneUTXOSetV1(bc.UTXO),
		bc.utxoOriginsV1(),
		make(map[string]bool),
		make(map[string]bool),
		known,
		checkPending,
		spendHeight,
	)
}

func (bc *Blockchain) ValidateTransactionUTXOV1(
	tx *Transaction,
	checkPending bool,
) (UTXOValidationResultV1, error) {
	return bc.ValidateTransactionUTXOAtHeightV1(
		tx,
		checkPending,
		bc.nextSpendHeightV1(),
	)
}

func (bc *Blockchain) ValidateTransactionsUTXOAtHeightV1(
	transactions []*Transaction,
	spendHeight int,
) (int, error) {
	if bc == nil {
		return 0, fmt.Errorf("%s: nil blockchain", UTXOConsensusVersionV1)
	}
	if spendHeight < 0 {
		return 0, fmt.Errorf("%s: invalid candidate spend height %d", UTXOConsensusVersionV1, spendHeight)
	}
	if bc.UTXO == nil {
		bc.UpdateUTXOSet()
	}

	scratch := cloneUTXOSetV1(bc.UTXO)
	origins := bc.utxoOriginsV1()
	createdNormalTx := make(map[string]bool)
	spentInSequence := make(map[string]bool)
	knownTxIDs := bc.knownTransactionIDsV1()
	totalFees := 0
	candidateBurnAtomsByOwner := make(map[string]int64)
	var candidateNetworkBurnAtoms int64

	for txIndex, tx := range transactions {
		if tx == nil {
			return 0, fmt.Errorf("%s: nil transaction at index %d", UTXOConsensusVersionV1, txIndex)
		}
		if tx.IsCoinbase() {
			txIDHex := hex.EncodeToString(tx.ID)
			if txIDHex == "" {
				return 0, fmt.Errorf("%s: coinbase at index %d has no id", UTXOConsensusVersionV1, txIndex)
			}
			if knownTxIDs[txIDHex] || createdNormalTx[txIDHex] {
				return 0, fmt.Errorf(
					"%s: duplicate transaction id: %s",
					UTXOConsensusVersionV1,
					txIDHex,
				)
			}
			knownTxIDs[txIDHex] = true
			continue
		}

		classification, err :=
			ClassifyCanonicalBurnTransactionAtHeightV1(
				tx,
				spendHeight,
				bc,
			)
		if err != nil {
			return 0, fmt.Errorf(
				"%s: tx[%d]: classify canonical burn: %w",
				UTXOConsensusVersionV1,
				txIndex,
				err,
			)
		}

		var burnOwnerMapKey string

		if classification.IsBurn {
			ownerKey, err :=
				bc.resolveCanonicalBurnOwnerKeyAgainstUTXOStateAtHeightV1(
					tx,
					scratch,
					spendHeight,
				)
			if err != nil {
				return 0, fmt.Errorf(
					"%s: tx[%d]: resolve canonical burn owner: %w",
					UTXOConsensusVersionV1,
					txIndex,
					err,
				)
			}

			burnOwnerMapKey = string(ownerKey)
			candidatePriorOwnerBurnAtoms :=
				candidateBurnAtomsByOwner[burnOwnerMapKey]

			if err :=
				bc.validateCanonicalBurnLimitsAtHeightV1(
					tx,
					spendHeight,
					ownerKey,
					candidatePriorOwnerBurnAtoms,
					candidateNetworkBurnAtoms,
				); err != nil {
				return 0, fmt.Errorf(
					"%s: tx[%d]: canonical burn limits: %w",
					UTXOConsensusVersionV1,
					txIndex,
					err,
				)
			}
		}

		result, err := bc.validateTransactionAgainstUTXOStateV1(
			tx,
			scratch,
			origins,
			createdNormalTx,
			spentInSequence,
			knownTxIDs,
			false,
			spendHeight,
		)
		if err != nil {
			return 0, fmt.Errorf("%s: tx[%d]: %w", UTXOConsensusVersionV1, txIndex, err)
		}

		if classification.IsBurn {
			ownerTotal :=
				candidateBurnAtomsByOwner[burnOwnerMapKey]
			nextOwnerTotal :=
				ownerTotal + classification.BurnAmountAtoms
			if nextOwnerTotal < ownerTotal {
				return 0, fmt.Errorf(
					"%s: tx[%d]: candidate owner burn total overflow",
					UTXOConsensusVersionV1,
					txIndex,
				)
			}

			nextNetworkTotal :=
				candidateNetworkBurnAtoms +
					classification.BurnAmountAtoms
			if nextNetworkTotal < candidateNetworkBurnAtoms {
				return 0, fmt.Errorf(
					"%s: tx[%d]: candidate network burn total overflow",
					UTXOConsensusVersionV1,
					txIndex,
				)
			}

			candidateBurnAtomsByOwner[burnOwnerMapKey] =
				nextOwnerTotal
			candidateNetworkBurnAtoms = nextNetworkTotal
		}

		nextFees := totalFees + result.Fee
		if nextFees < totalFees {
			return 0, fmt.Errorf("%s: block fee total overflow", UTXOConsensusVersionV1)
		}
		totalFees = nextFees
	}

	return totalFees, nil
}

func (bc *Blockchain) ValidateTransactionsUTXOV1(
	transactions []*Transaction,
) (int, error) {
	return bc.ValidateTransactionsUTXOAtHeightV1(
		transactions,
		bc.nextSpendHeightV1(),
	)
}
