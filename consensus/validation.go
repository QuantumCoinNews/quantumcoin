package consensus

import "fmt"

const ConsensusValidationVersionV1 = "QC_CONSENSUS_VALIDATION_V1"

// CanonicalBlockValidationInputV1 contains the stateless consensus data
// required to validate one canonical block envelope.
//
// Chain-state-dependent checks remain in the blockchain package, including:
//   - previous-block linkage
//   - expected dynamic difficulty
//   - UTXO ownership and value conservation
//   - coinbase maturity
//   - founder vesting
//   - replacement-chain eligibility
type CanonicalBlockValidationInputV1 struct {
	Protocol ProtocolVersion
	Height   uint64

	PoWHeader PoWHeaderV1

	PoWHeaderV4 PoWHeaderV4
	Nonce       uint64
	StoredHash  []byte

	Transactions []*CoinbaseTransactionViewV1

	ExpectedSubsidy int64
	VerifiedFees    int64
	TotalSupply     int64
}

// CanonicalBlockValidationResultV1 returns validated stateless consensus
// results without mutating caller-owned state.
type CanonicalBlockValidationResultV1 struct {
	Protocol ProtocolVersion
	Height   uint64

	Coinbase CoinbaseLayoutResultV1
	Reward   BlockRewardBreakdownV1
}

// ValidateCanonicalBlockV1 is the canonical stateless validation entrypoint.
//
// Validation order:
//  1. protocol selection
//  2. stored PoW hash and target
//  3. coinbase layout
//  4. deterministic subsidy and verified-fee equation
//
// ContinuityV2 remains fail-closed until fully implemented.
func ValidateCanonicalBlockV1(
	input CanonicalBlockValidationInputV1,
) (CanonicalBlockValidationResultV1, error) {
	result := CanonicalBlockValidationResultV1{
		Protocol: input.Protocol,
		Height:   input.Height,
	}

	switch input.Protocol {
	case ProtocolLegacyV1:
	// LegacyV1 validation continues below.

	case ProtocolAtomV3, ProtocolNFTV4:
		if err := ValidateProtocolAmountRuntimeV1(
			input.Protocol,
		); err != nil {
			return CanonicalBlockValidationResultV1{}, fmt.Errorf(
				"%s: protocol runtime validation failed: %w",
				ConsensusValidationVersionV1,
				err,
			)
		}

	case ProtocolContinuityV2:
		return CanonicalBlockValidationResultV1{}, fmt.Errorf(
			"%s: protocol %q is not implemented",
			ConsensusValidationVersionV1,
			input.Protocol,
		)

	default:
		return CanonicalBlockValidationResultV1{}, fmt.Errorf(
			"%s: unsupported protocol %q",
			ConsensusValidationVersionV1,
			input.Protocol,
		)
	}

	var powValidationErr error
	switch input.Protocol {
	case ProtocolNFTV4:
		powValidationErr = ValidateStoredPoWHashV4(
			input.PoWHeaderV4,
			input.Nonce,
			input.StoredHash,
		)
	default:
		powValidationErr = ValidateStoredPoWHashV1(
			input.PoWHeader,
			input.Nonce,
			input.StoredHash,
		)
	}
	if powValidationErr != nil {
		return CanonicalBlockValidationResultV1{}, fmt.Errorf(
			"%s: proof-of-work validation failed: %w",
			ConsensusValidationVersionV1,
			powValidationErr,
		)
	}

	coinbaseResult, err := ValidateCoinbaseLayoutV1(
		input.Transactions,
		input.Height == 0,
	)
	if err != nil {
		return CanonicalBlockValidationResultV1{}, fmt.Errorf(
			"%s: coinbase layout validation failed: %w",
			ConsensusValidationVersionV1,
			err,
		)
	}

	rewardResult, err := ValidateBlockRewardV1(
		input.Height,
		coinbaseResult.Total,
		input.ExpectedSubsidy,
		input.VerifiedFees,
		input.TotalSupply,
	)
	if err != nil {
		return CanonicalBlockValidationResultV1{}, fmt.Errorf(
			"%s: block reward validation failed: %w",
			ConsensusValidationVersionV1,
			err,
		)
	}

	result.Coinbase = coinbaseResult
	result.Reward = rewardResult

	return result, nil
}
