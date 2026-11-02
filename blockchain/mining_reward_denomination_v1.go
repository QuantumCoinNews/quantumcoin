package blockchain

import (
	"fmt"
	"math"

	qconsensus "quantumcoin/consensus"
)

const MiningRewardDenominationVersionV1 = "QC_MINING_REWARD_DENOMINATION_V1"

// ComposeMiningRewardUnitsV1 combines subsidy and verified fees only after both
// values are expressed in the transaction denomination of the target protocol.
func ComposeMiningRewardUnitsV1(
	protocol qconsensus.ProtocolVersion,
	height int,
	totalMintedQC int,
	totalSupplyQC int,
	verifiedFees int,
) (subsidyUnits int, coinbaseTotal int, err error) {
	if verifiedFees < 0 {
		return 0, 0, fmt.Errorf(
			"%s: verified fees cannot be negative: %d",
			MiningRewardDenominationVersionV1,
			verifiedFees,
		)
	}

	subsidyUnits, err =
		ExpectedSubsidyUnitsFromStateV1(
			protocol,
			height,
			totalMintedQC,
			totalSupplyQC,
		)
	if err != nil {
		return 0, 0, fmt.Errorf(
			"%s: subsidy units: %w",
			MiningRewardDenominationVersionV1,
			err,
		)
	}

	if subsidyUnits <= 0 {
		return subsidyUnits, 0, nil
	}

	if verifiedFees > math.MaxInt-subsidyUnits {
		return 0, 0, fmt.Errorf(
			"%s: subsidy plus fees overflow",
			MiningRewardDenominationVersionV1,
		)
	}

	return subsidyUnits, subsidyUnits + verifiedFees, nil
}

// miningRewardUnitsForHeightV1 resolves the protocol and current QC emission
// state, then composes the transaction-wire coinbase amount.
func (bc *Blockchain) miningRewardUnitsForHeightV1(
	height int,
	verifiedFees int,
) (int, int, error) {
	if bc == nil {
		return 0, 0, fmt.Errorf(
			"%s: nil blockchain",
			MiningRewardDenominationVersionV1,
		)
	}

	return ComposeMiningRewardUnitsV1(
		bc.ProtocolAtHeightV1(height),
		height,
		bc.TotalMinted(),
		bc.TotalSupply,
		verifiedFees,
	)
}
