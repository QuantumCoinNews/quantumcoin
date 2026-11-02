package blockchain

import (
	"fmt"
	"math"

	qconsensus "quantumcoin/consensus"
)

const ReplacementMintedDenominationVersionV1 = "QC_REPLACEMENT_MINTED_DENOMINATION_V1"

// ProtocolSubsidyUnitsToWholeQCV1 converts a validated block subsidy from the
// transaction/consensus denomination used by the selected protocol back into
// the whole-QC policy unit used by emission accounting.
//
// LegacyV1 and ContinuityV2 already carry whole-QC integer subsidy values.
// AtomV3 and NFTV4 carry atoms and require an exact whole-QC subsidy because the
// current canonical emission schedule is defined in whole QC.
func ProtocolSubsidyUnitsToWholeQCV1(
	protocol qconsensus.ProtocolVersion,
	subsidyUnits int64,
) (int, error) {
	if subsidyUnits < 0 {
		return 0, fmt.Errorf(
			"%s: negative subsidy units: %d",
			ReplacementMintedDenominationVersionV1,
			subsidyUnits,
		)
	}

	var subsidyQC int64

	switch protocol {
	case qconsensus.ProtocolLegacyV1,
		qconsensus.ProtocolContinuityV2:
		subsidyQC = subsidyUnits

	case qconsensus.ProtocolAtomV3, qconsensus.ProtocolNFTV4:
		if err := qconsensus.ValidateProtocolAmountRuntimeV1(
			protocol,
		); err != nil {
			return 0, fmt.Errorf(
				"%s: atom runtime: %w",
				ReplacementMintedDenominationVersionV1,
				err,
			)
		}

		if subsidyUnits%
			qconsensus.CanonicalAtomsPerQCV1 != 0 {
			return 0, fmt.Errorf(
				"%s: atom subsidy is not an exact whole-QC amount: %d atoms",
				ReplacementMintedDenominationVersionV1,
				subsidyUnits,
			)
		}

		subsidyQC =
			subsidyUnits /
				qconsensus.CanonicalAtomsPerQCV1

	default:
		return 0, fmt.Errorf(
			"%s: unsupported protocol %q",
			ReplacementMintedDenominationVersionV1,
			protocol,
		)
	}

	if subsidyQC > int64(math.MaxInt) {
		return 0, fmt.Errorf(
			"%s: whole-QC subsidy exceeds runtime int capacity: %d",
			ReplacementMintedDenominationVersionV1,
			subsidyQC,
		)
	}

	return int(subsidyQC), nil
}
