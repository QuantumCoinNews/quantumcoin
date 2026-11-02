package blockchain

import (
	"fmt"

	qconsensus "quantumcoin/consensus"
)

const CanonicalBlockValidationAdapterVersionV1 = "QC_CANONICAL_BLOCK_VALIDATION_ADAPTER_V1"

// ValidateCanonicalBlockEnvelopeV1 converts the historical blockchain Block
// model into the chain-independent consensus validation input.
//
// Chain-state-dependent values are resolved before invoking the stateless
// consensus validator:
//
//   - protocol at candidate height
//   - verified transaction fees
//   - expected subsidy from minted supply state
//
// Previous-hash linkage, expected dynamic difficulty, founder vesting and
// replacement-chain eligibility remain in their existing validation paths.
func (bc *Blockchain) ValidateCanonicalBlockEnvelopeV1(
	block *Block,
	totalMintedBefore int,
) (qconsensus.CanonicalBlockValidationResultV1, error) {
	var zero qconsensus.CanonicalBlockValidationResultV1

	if bc == nil {
		return zero, fmt.Errorf(
			"%s: nil blockchain",
			CanonicalBlockValidationAdapterVersionV1,
		)
	}

	if block == nil {
		return zero, ErrNilBlock
	}

	if block.Index < 0 {
		return zero, fmt.Errorf(
			"%s: negative block height: %d",
			CanonicalBlockValidationAdapterVersionV1,
			block.Index,
		)
	}

	if block.Nonce < 0 {
		return zero, fmt.Errorf(
			"%s: negative block nonce: %d",
			CanonicalBlockValidationAdapterVersionV1,
			block.Nonce,
		)
	}

	protocol := bc.ProtocolAtHeightV1(block.Index)
	if err := validateNFTMintUniquenessV1(bc.Blocks, block); err != nil {
		return zero, fmt.Errorf(
			"%s: %w",
			CanonicalBlockValidationAdapterVersionV1,
			err,
		)
	}
	if err := bc.validateNFTMintWithLocalContextV1(bc.Blocks, block); err != nil {
		return zero, fmt.Errorf(
			"%s: %w",
			CanonicalBlockValidationAdapterVersionV1,
			err,
		)
	}
	var powHeaderV1 qconsensus.PoWHeaderV1
	var powHeaderV4 qconsensus.PoWHeaderV4
	var err error

	if protocol == qconsensus.ProtocolNFTV4 {
		powHeaderV4, err = canonicalPoWHeaderV4(block)
	} else {
		powHeaderV1, err = canonicalPoWHeaderV1(block)
	}
	if err != nil {
		return zero, fmt.Errorf(
			"%s: canonical PoW header: %w",
			CanonicalBlockValidationAdapterVersionV1,
			err,
		)
	}

	transactionViews := make(
		[]*qconsensus.CoinbaseTransactionViewV1,
		len(block.Transactions),
	)

	for transactionIndex, transaction := range block.Transactions {
		if transaction == nil {
			transactionViews[transactionIndex] = nil
			continue
		}

		outputAmounts := make(
			[]int64,
			len(transaction.Outputs),
		)

		for outputIndex, output := range transaction.Outputs {
			outputAmounts[outputIndex] =
				int64(output.Amount)
		}

		transactionViews[transactionIndex] =
			&qconsensus.CoinbaseTransactionViewV1{
				Coinbase:      transaction.IsCoinbase(),
				OutputAmounts: outputAmounts,
			}
	}

	totalSupplyUnits, err :=
		qconsensus.WholeQCAmountToProtocolUnitsV1(
			protocol,
			int64(bc.TotalSupply),
		)
	if err != nil {
		return zero, fmt.Errorf(
			"%s: total supply denomination: %w",
			CanonicalBlockValidationAdapterVersionV1,
			err,
		)
	}

	expectedSubsidy := 0
	verifiedFees := 0

	if block.Index > 0 {
		verifiedFees, err =
			bc.CalculateBlockFeesAtHeightV1(
				block.Transactions,
				block.Index,
			)
		if err != nil {
			return zero, fmt.Errorf(
				"%s: verified transaction fees: %w",
				CanonicalBlockValidationAdapterVersionV1,
				err,
			)
		}

		expectedSubsidy, err =
			ExpectedSubsidyUnitsFromStateV1(
				protocol,
				block.Index,
				totalMintedBefore,
				bc.TotalSupply,
			)
		if err != nil {
			return zero, fmt.Errorf(
				"%s: expected subsidy denomination: %w",
				CanonicalBlockValidationAdapterVersionV1,
				err,
			)
		}
	}

	result, err := qconsensus.ValidateCanonicalBlockV1(
		qconsensus.CanonicalBlockValidationInputV1{
			Protocol: protocol,
			Height:   uint64(block.Index),

			PoWHeader:   powHeaderV1,
			PoWHeaderV4: powHeaderV4,
			Nonce:       uint64(block.Nonce),
			StoredHash: append(
				[]byte(nil),
				block.Hash...,
			),

			Transactions: transactionViews,

			ExpectedSubsidy: int64(
				expectedSubsidy,
			),
			VerifiedFees: int64(
				verifiedFees,
			),
			TotalSupply: totalSupplyUnits,
		},
	)
	if err != nil {
		return zero, fmt.Errorf(
			"%s: %w",
			CanonicalBlockValidationAdapterVersionV1,
			err,
		)
	}

	return result, nil
}
