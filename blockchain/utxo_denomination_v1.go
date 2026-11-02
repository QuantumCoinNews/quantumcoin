package blockchain

import (
	"fmt"
	"math"

	qconsensus "quantumcoin/consensus"
)

const UTXODenominationVersionV1 = "QC_UTXO_DENOMINATION_V1"

// normalizeUTXOAmountForSpendV1 converts one historical UTXO amount into the
// monetary integer unit required at spendHeight.
//
// Historical TransactionOutput values are intentionally not rewritten:
//
//   - LegacyV1 / ContinuityV2 outputs store whole-QC integers.
//   - AtomV3 outputs store atom integers.
//
// When a pre-Atom UTXO is spent under AtomV3, its raw whole-QC amount is
// deterministically converted to atoms.
//
// This preserves chain history while preventing a legacy 50 QC output from
// being misinterpreted as 50 atoms after AtomV3 activation.
func (bc *Blockchain) normalizeUTXOAmountForSpendV1(
	rawAmount int,
	originHeight int,
	spendHeight int,
) (int, error) {
	if bc == nil {
		return 0, fmt.Errorf(
			"%s: nil blockchain",
			UTXODenominationVersionV1,
		)
	}

	if rawAmount <= 0 {
		return 0, fmt.Errorf(
			"%s: amount must be positive",
			UTXODenominationVersionV1,
		)
	}

	if originHeight < 0 {
		return 0, fmt.Errorf(
			"%s: invalid origin height %d",
			UTXODenominationVersionV1,
			originHeight,
		)
	}

	if spendHeight < 0 {
		return 0, fmt.Errorf(
			"%s: invalid spend height %d",
			UTXODenominationVersionV1,
			spendHeight,
		)
	}

	if originHeight > spendHeight {
		return 0, fmt.Errorf(
			"%s: origin height %d exceeds spend height %d",
			UTXODenominationVersionV1,
			originHeight,
			spendHeight,
		)
	}

	originProtocol := bc.ProtocolAtHeightV1(originHeight)
	spendProtocol := bc.ProtocolAtHeightV1(spendHeight)

	originUnit, err :=
		qconsensus.MonetaryUnitAtProtocolV1(
			originProtocol,
		)
	if err != nil {
		return 0, fmt.Errorf(
			"%s: origin protocol denomination: %w",
			UTXODenominationVersionV1,
			err,
		)
	}

	spendUnit, err :=
		qconsensus.MonetaryUnitAtProtocolV1(
			spendProtocol,
		)
	if err != nil {
		return 0, fmt.Errorf(
			"%s: spend protocol denomination: %w",
			UTXODenominationVersionV1,
			err,
		)
	}

	if originUnit == spendUnit {
		return rawAmount, nil
	}

	if originUnit == qconsensus.MonetaryUnitWholeQCV1 &&
		spendUnit == qconsensus.MonetaryUnitAtomV1 {

		if err := qconsensus.ValidateProtocolAmountRuntimeV1(
			spendProtocol,
		); err != nil {
			return 0, fmt.Errorf(
				"%s: %w",
				UTXODenominationVersionV1,
				err,
			)
		}

		atoms, err :=
			qconsensus.WholeQCToAtomsV1(
				int64(rawAmount),
			)
		if err != nil {
			return 0, fmt.Errorf(
				"%s: legacy QC to atom conversion: %w",
				UTXODenominationVersionV1,
				err,
			)
		}

		if atoms > int64(math.MaxInt) {
			return 0, fmt.Errorf(
				"%s: atom amount exceeds runtime int capacity",
				UTXODenominationVersionV1,
			)
		}

		return int(atoms), nil
	}

	return 0, fmt.Errorf(
		"%s: unsupported denomination transition %q -> %q",
		UTXODenominationVersionV1,
		originUnit,
		spendUnit,
	)
}
