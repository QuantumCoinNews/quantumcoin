package consensus

import "testing"

func TestAtomScaleBlockRewardEquationV1(
	t *testing.T,
) {
	result, err := ValidateBlockRewardV1(
		1,
		50_000_001,
		50_000_000,
		1,
		25_500_000_000_000,
	)
	if err != nil {
		t.Fatalf(
			"Atom-scale reward equation rejected: %v",
			err,
		)
	}

	if result.Subsidy != 50_000_000 {
		t.Fatalf(
			"subsidy=%d want=50000000",
			result.Subsidy,
		)
	}

	if result.Fees != 1 {
		t.Fatalf(
			"fees=%d want=1",
			result.Fees,
		)
	}

	if result.CoinbaseTotal != 50_000_001 {
		t.Fatalf(
			"coinbase=%d want=50000001",
			result.CoinbaseTotal,
		)
	}
}

func TestAtomScaleBlockRewardRejectsMixedUnitsV1(
	t *testing.T,
) {
	if _, err := ValidateBlockRewardV1(
		1,
		51,
		50_000_000,
		1,
		25_500_000_000_000,
	); err == nil {
		t.Fatal(
			"mixed whole-QC/atom reward unexpectedly accepted",
		)
	}
}

func TestAtomScaleCoinbaseLayoutUsesInt64WithoutScalingAssumptionsV1(
	t *testing.T,
) {
	txs := []*CoinbaseTransactionViewV1{
		{
			Coinbase: true,
			OutputAmounts: []int64{
				50_000_001,
			},
		},
	}

	result, err := ValidateCoinbaseLayoutV1(
		txs,
		false,
	)
	if err != nil {
		t.Fatalf(
			"Atom-scale coinbase layout rejected: %v",
			err,
		)
	}

	if result.Total != 50_000_001 {
		t.Fatalf(
			"coinbase total=%d want=50000001",
			result.Total,
		)
	}
}

func TestAtomScaleCoinbaseLayoutRejectsOverflowV1(
	t *testing.T,
) {
	txs := []*CoinbaseTransactionViewV1{
		{
			Coinbase: true,
			OutputAmounts: []int64{
				9223372036854775807,
				1,
			},
		},
	}

	if _, err := ValidateCoinbaseLayoutV1(
		txs,
		false,
	); err == nil {
		t.Fatal(
			"coinbase int64 overflow unexpectedly accepted",
		)
	}
}

func TestCanonicalValidatorAcceptsAtomV3EnvelopeV1(
	t *testing.T,
) {
	header, nonce, storedHash :=
		canonicalValidationPoWFixtureV1(t)

	input := CanonicalBlockValidationInputV1{
		Protocol: ProtocolAtomV3,
		Height:   1,

		PoWHeader:  header,
		Nonce:      nonce,
		StoredHash: storedHash,

		Transactions: []*CoinbaseTransactionViewV1{
			{
				Coinbase: true,
				OutputAmounts: []int64{
					50_000_001,
				},
			},
			{
				Coinbase: false,
			},
		},

		ExpectedSubsidy: 50_000_000,
		VerifiedFees:    1,
		TotalSupply:     25_500_000_000_000,
	}

	result, err := ValidateCanonicalBlockV1(input)
	if err != nil {
		t.Fatalf(
			"valid AtomV3 canonical envelope rejected: %v",
			err,
		)
	}

	if result.Protocol != ProtocolAtomV3 {
		t.Fatalf(
			"protocol=%q want=%q",
			result.Protocol,
			ProtocolAtomV3,
		)
	}

	if result.Reward.Subsidy != 50_000_000 {
		t.Fatalf(
			"subsidy=%d want=50000000",
			result.Reward.Subsidy,
		)
	}

	if result.Reward.Fees != 1 {
		t.Fatalf(
			"fees=%d want=1",
			result.Reward.Fees,
		)
	}

	if result.Reward.CoinbaseTotal != 50_000_001 {
		t.Fatalf(
			"coinbase=%d want=50000001",
			result.Reward.CoinbaseTotal,
		)
	}
}

func TestCanonicalValidatorAtomV3RejectsMixedRewardUnitsV1(
	t *testing.T,
) {
	header, nonce, storedHash :=
		canonicalValidationPoWFixtureV1(t)

	input := CanonicalBlockValidationInputV1{
		Protocol: ProtocolAtomV3,
		Height:   1,

		PoWHeader:  header,
		Nonce:      nonce,
		StoredHash: storedHash,

		Transactions: []*CoinbaseTransactionViewV1{
			{
				Coinbase: true,
				OutputAmounts: []int64{
					51,
				},
			},
		},

		ExpectedSubsidy: 50_000_000,
		VerifiedFees:    1,
		TotalSupply:     25_500_000_000_000,
	}

	if _, err := ValidateCanonicalBlockV1(
		input,
	); err == nil {
		t.Fatal(
			"mixed-unit AtomV3 canonical reward unexpectedly accepted",
		)
	}
}
