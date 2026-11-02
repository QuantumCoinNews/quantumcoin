package consensus

import (
	"crypto/sha256"
	"strings"
	"testing"
)

func canonicalValidationPoWFixtureV1(
	t *testing.T,
) (PoWHeaderV1, uint64, []byte) {
	t.Helper()

	transactionDigest := sha256.Sum256(
		[]byte("quantumcoin-validation-v1"),
	)

	header := PoWHeaderV1{
		PreviousHash:      []byte("previous-block-hash"),
		TransactionDigest: transactionDigest[:],
		Timestamp:         1_725_158_430,
		Difficulty:        1,
	}

	for nonce := uint64(0); nonce <= MaxConsensusNonceV1; nonce++ {
		meetsTarget, hash, err := PoWHashMeetsTargetV1(
			header,
			nonce,
		)
		if err != nil {
			t.Fatalf(
				"calculate validation fixture PoW: %v",
				err,
			)
		}

		if meetsTarget {
			return header,
				nonce,
				append([]byte(nil), hash[:]...)
		}

		if nonce == MaxConsensusNonceV1 {
			break
		}
	}

	t.Fatal("validation fixture nonce was not found")
	return PoWHeaderV1{}, 0, nil
}

func canonicalValidationInputV1(
	t *testing.T,
) CanonicalBlockValidationInputV1 {
	t.Helper()

	header, nonce, storedHash :=
		canonicalValidationPoWFixtureV1(t)

	return CanonicalBlockValidationInputV1{
		Protocol: ProtocolLegacyV1,
		Height:   1,

		PoWHeader:  header,
		Nonce:      nonce,
		StoredHash: storedHash,

		Transactions: []*CoinbaseTransactionViewV1{
			{
				Coinbase: true,
				OutputAmounts: []int64{
					53,
				},
			},
			{
				Coinbase: false,
			},
		},

		ExpectedSubsidy: 50,
		VerifiedFees:    3,
		TotalSupply:     25_500_000,
	}
}

func TestValidateCanonicalBlockV1AcceptsLegacyBlock(
	t *testing.T,
) {
	input := canonicalValidationInputV1(t)

	result, err := ValidateCanonicalBlockV1(input)
	if err != nil {
		t.Fatalf(
			"valid canonical block rejected: %v",
			err,
		)
	}

	if result.Protocol != ProtocolLegacyV1 {
		t.Fatalf(
			"protocol = %q, want %q",
			result.Protocol,
			ProtocolLegacyV1,
		)
	}

	if result.Height != 1 {
		t.Fatalf(
			"height = %d, want 1",
			result.Height,
		)
	}

	if result.Coinbase.Count != 1 ||
		result.Coinbase.Total != 53 {
		t.Fatalf(
			"unexpected coinbase result: %+v",
			result.Coinbase,
		)
	}

	if result.Reward.Subsidy != 50 ||
		result.Reward.Fees != 3 ||
		result.Reward.CoinbaseTotal != 53 {
		t.Fatalf(
			"unexpected reward result: %+v",
			result.Reward,
		)
	}
}

func TestValidateCanonicalBlockV1RejectsStoredHashMismatch(
	t *testing.T,
) {
	input := canonicalValidationInputV1(t)
	input.StoredHash = make([]byte, sha256.Size)

	_, err := ValidateCanonicalBlockV1(input)
	if err == nil {
		t.Fatal(
			"stored PoW hash mismatch was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"proof-of-work validation failed",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestValidateCanonicalBlockV1RejectsInvalidCoinbaseLayout(
	t *testing.T,
) {
	input := canonicalValidationInputV1(t)

	input.Transactions = []*CoinbaseTransactionViewV1{
		{
			Coinbase: false,
		},
		{
			Coinbase: true,
			OutputAmounts: []int64{
				53,
			},
		},
	}

	_, err := ValidateCanonicalBlockV1(input)
	if err == nil {
		t.Fatal(
			"invalid coinbase layout was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"coinbase layout validation failed",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestValidateCanonicalBlockV1RejectsRewardMismatch(
	t *testing.T,
) {
	input := canonicalValidationInputV1(t)
	input.ExpectedSubsidy = 51

	_, err := ValidateCanonicalBlockV1(input)
	if err == nil {
		t.Fatal(
			"invalid block reward was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"block reward validation failed",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestValidateCanonicalBlockV1RejectsUnimplementedContinuityV2(
	t *testing.T,
) {
	input := canonicalValidationInputV1(t)
	input.Protocol = ProtocolContinuityV2

	_, err := ValidateCanonicalBlockV1(input)
	if err == nil {
		t.Fatal(
			"unimplemented continuity protocol was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"not implemented",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}
