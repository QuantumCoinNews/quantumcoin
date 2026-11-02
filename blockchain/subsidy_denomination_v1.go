package blockchain

import (
	"fmt"
	"math"

	qconsensus "quantumcoin/consensus"
)

const SubsidyDenominationVersionV1 = "QC_SUBSIDY_DENOMINATION_V1"

// ExpectedSubsidyUnitsFromStateV1 keeps emission accounting in whole QC,
// then converts the resulting subsidy into the integer denomination carried
// by transactions at the selected protocol.
//
// totalMinted and totalSupply remain whole-QC policy values.
func ExpectedSubsidyUnitsFromStateV1(
	protocol qconsensus.ProtocolVersion,
	height int,
	totalMintedQC int,
	totalSupplyQC int,
) (int, error) {
	subsidyQC := ExpectedSubsidyFromStateV1(
		height,
		totalMintedQC,
		totalSupplyQC,
	)
	if subsidyQC <= 0 {
		return 0, nil
	}

	units, err :=
		qconsensus.WholeQCAmountToProtocolUnitsV1(
			protocol,
			int64(subsidyQC),
		)
	if err != nil {
		return 0, fmt.Errorf(
			"%s: convert subsidy to protocol units: %w",
			SubsidyDenominationVersionV1,
			err,
		)
	}

	if units > int64(math.MaxInt) {
		return 0, fmt.Errorf(
			"%s: subsidy exceeds runtime int capacity",
			SubsidyDenominationVersionV1,
		)
	}

	return int(units), nil
}

// ExpectedSubsidyUnitsForHeightV1 resolves the protocol active at height and
// returns the transaction-wire subsidy in that protocol's denomination.
func (bc *Blockchain) ExpectedSubsidyUnitsForHeightV1(
	height int,
) (int, error) {
	if bc == nil {
		return 0, fmt.Errorf(
			"%s: nil blockchain",
			SubsidyDenominationVersionV1,
		)
	}

	protocol := bc.ProtocolAtHeightV1(height)

	return ExpectedSubsidyUnitsFromStateV1(
		protocol,
		height,
		bc.TotalMinted(),
		bc.TotalSupply,
	)
}
