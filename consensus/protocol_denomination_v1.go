package consensus

import (
	"fmt"
	"strconv"
)

const ProtocolDenominationVersionV1 = "QC_PROTOCOL_DENOMINATION_V1"

// MonetaryUnitV1 identifies the integer unit carried by consensus amount
// fields at a particular protocol version.
type MonetaryUnitV1 string

const (
	MonetaryUnitWholeQCV1 MonetaryUnitV1 = "QC_WHOLE_QC_INTEGER"

	MonetaryUnitAtomV1 MonetaryUnitV1 = "QC_ATOM_INTEGER"
)

// ProtocolUsesAtomDenominationV1 reports whether monetary integer fields are
// interpreted as atoms rather than whole QC.
//
// It does not activate any protocol.
func ProtocolUsesAtomDenominationV1(
	protocol ProtocolVersion,
) bool {
	return protocol == ProtocolAtomV3 || protocol == ProtocolNFTV4
}

// MonetaryUnitAtProtocolV1 returns the denomination semantics for consensus
// monetary integer fields.
func MonetaryUnitAtProtocolV1(
	protocol ProtocolVersion,
) (MonetaryUnitV1, error) {
	switch protocol {
	case ProtocolLegacyV1, ProtocolContinuityV2:
		return MonetaryUnitWholeQCV1, nil

	case ProtocolAtomV3, ProtocolNFTV4:
		return MonetaryUnitAtomV1, nil

	default:
		return "", fmt.Errorf(
			"%s: unsupported protocol %q",
			ProtocolDenominationVersionV1,
			protocol,
		)
	}
}

// ValidateProtocolAmountRuntimeV1 protects the current historical Transaction
// wire model, whose Amount fields use Go int.
//
// AtomV3 requires 64-bit int capacity because the canonical atom-denominated
// total supply exceeds the signed 32-bit integer range.
//
// Legacy transaction serialization remains unchanged.
func ValidateProtocolAmountRuntimeV1(
	protocol ProtocolVersion,
) error {
	if !ProtocolUsesAtomDenominationV1(protocol) {
		return nil
	}

	if strconv.IntSize != 64 {
		return fmt.Errorf(
			"%s: AtomV3 requires 64-bit int runtime",
			ProtocolDenominationVersionV1,
		)
	}

	return nil
}

// WholeQCAmountToProtocolUnitsV1 converts a non-negative whole-QC consensus
// value into the integer denomination used by the selected protocol.
func WholeQCAmountToProtocolUnitsV1(
	protocol ProtocolVersion,
	qc int64,
) (int64, error) {
	if qc < 0 {
		return 0, ErrInvalidQCAmountV1
	}

	switch protocol {
	case ProtocolLegacyV1, ProtocolContinuityV2:
		return qc, nil

	case ProtocolAtomV3, ProtocolNFTV4:
		if err := ValidateProtocolAmountRuntimeV1(
			protocol,
		); err != nil {
			return 0, err
		}

		return WholeQCToAtomsV1(qc)

	default:
		return 0, fmt.Errorf(
			"%s: unsupported protocol %q",
			ProtocolDenominationVersionV1,
			protocol,
		)
	}
}
