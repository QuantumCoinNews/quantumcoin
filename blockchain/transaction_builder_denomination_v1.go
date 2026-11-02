package blockchain

import (
	"fmt"
	"math"

	qconsensus "quantumcoin/consensus"
)

const TransactionBuilderDenominationVersionV1 = "QC_TRANSACTION_BUILDER_DENOMINATION_V1"

// wholeQCAmountToTransactionUnitsAtHeightV1 converts a legacy/user-facing
// whole-QC amount into the integer monetary unit required at spendHeight.
func (bc *Blockchain) wholeQCAmountToTransactionUnitsAtHeightV1(
	wholeQC int,
	spendHeight int,
) (int, error) {
	if bc == nil {
		return 0, fmt.Errorf(
			"%s: nil blockchain",
			TransactionBuilderDenominationVersionV1,
		)
	}

	if wholeQC <= 0 {
		return 0, fmt.Errorf(
			"%s: whole-QC amount must be positive",
			TransactionBuilderDenominationVersionV1,
		)
	}

	if spendHeight < 0 {
		return 0, fmt.Errorf(
			"%s: invalid spend height %d",
			TransactionBuilderDenominationVersionV1,
			spendHeight,
		)
	}

	protocol := bc.ProtocolAtHeightV1(spendHeight)

	units, err :=
		qconsensus.WholeQCAmountToProtocolUnitsV1(
			protocol,
			int64(wholeQC),
		)
	if err != nil {
		return 0, fmt.Errorf(
			"%s: convert whole QC for protocol %q: %w",
			TransactionBuilderDenominationVersionV1,
			protocol,
			err,
		)
	}

	if units <= 0 {
		return 0, fmt.Errorf(
			"%s: converted amount must be positive",
			TransactionBuilderDenominationVersionV1,
		)
	}

	if units > int64(math.MaxInt) {
		return 0, fmt.Errorf(
			"%s: converted amount exceeds runtime int capacity",
			TransactionBuilderDenominationVersionV1,
		)
	}

	return int(units), nil
}

// qcAmountStringToTransactionUnitsAtHeightV1 parses an exact user-facing
// QC decimal string and converts it to the integer denomination required
// by spendHeight. No floating-point conversion is used.
func (bc *Blockchain) qcAmountStringToTransactionUnitsAtHeightV1(
	rawQC string,
	spendHeight int,
) (int, error) {
	if bc == nil {
		return 0, fmt.Errorf(
			"%s: nil blockchain",
			TransactionBuilderDenominationVersionV1,
		)
	}

	if spendHeight < 0 {
		return 0, fmt.Errorf(
			"%s: invalid spend height %d",
			TransactionBuilderDenominationVersionV1,
			spendHeight,
		)
	}

	atoms, err := qconsensus.ParseQCToAtomsV1(rawQC)
	if err != nil {
		return 0, fmt.Errorf(
			"%s: invalid QC amount: %w",
			TransactionBuilderDenominationVersionV1,
			err,
		)
	}

	if atoms <= 0 {
		return 0, fmt.Errorf(
			"%s: QC amount must be positive",
			TransactionBuilderDenominationVersionV1,
		)
	}

	protocol := bc.ProtocolAtHeightV1(spendHeight)

	var units int64

	switch protocol {
	case qconsensus.ProtocolLegacyV1:
		if atoms%qconsensus.CanonicalAtomsPerQCV1 != 0 {
			return 0, fmt.Errorf(
				"%s: fractional QC is not supported by LegacyV1",
				TransactionBuilderDenominationVersionV1,
			)
		}

		units = atoms / qconsensus.CanonicalAtomsPerQCV1

	case qconsensus.ProtocolAtomV3, qconsensus.ProtocolNFTV4:
		if err := qconsensus.ValidateProtocolAmountRuntimeV1(
			protocol,
		); err != nil {
			return 0, fmt.Errorf(
				"%s: atom protocol runtime validation: %w",
				TransactionBuilderDenominationVersionV1,
				err,
			)
		}

		units = atoms

	case qconsensus.ProtocolContinuityV2:
		return 0, fmt.Errorf(
			"%s: ContinuityV2 remains disabled",
			TransactionBuilderDenominationVersionV1,
		)

	default:
		return 0, fmt.Errorf(
			"%s: unsupported protocol %q",
			TransactionBuilderDenominationVersionV1,
			protocol,
		)
	}

	if units <= 0 || units > int64(math.MaxInt) {
		return 0, fmt.Errorf(
			"%s: protocol amount exceeds runtime int capacity",
			TransactionBuilderDenominationVersionV1,
		)
	}

	return int(units), nil
}
