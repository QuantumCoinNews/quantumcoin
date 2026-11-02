package consensus

import "testing"

func atomActivationTestPlanV1() ActivationPlan {
	return ActivationPlan{
		NetworkID:        "testnet",
		LegacyGenesisHex: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ActivationHeight: 100,
		PreserveHistory:  true,
		PreserveBalances: true,
	}
}

func TestEmptyActivationTargetPreservesContinuityV2V1(t *testing.T) {
	plan := atomActivationTestPlanV1()

	if err := plan.Validate(); err != nil {
		t.Fatalf("legacy-compatible activation plan: %v", err)
	}

	if got := plan.EffectiveTargetProtocol(); got != ProtocolContinuityV2 {
		t.Fatalf(
			"effective target=%q want=%q",
			got,
			ProtocolContinuityV2,
		)
	}

	if got := ProtocolAtHeight(100, 100); got != ProtocolContinuityV2 {
		t.Fatalf(
			"historical selector=%q want=%q",
			got,
			ProtocolContinuityV2,
		)
	}
}

func TestActivationPlanCanExplicitlyTargetAtomV3V1(t *testing.T) {
	plan := atomActivationTestPlanV1()
	plan.TargetProtocol = ProtocolAtomV3

	if err := plan.Validate(); err != nil {
		t.Fatalf("atom activation plan: %v", err)
	}

	policy := ActivationPolicyV1{
		Enabled: true,
		Plan:    plan,
	}

	policy = verifiedActivationFixtureV1(t, policy)
	if err := policy.Validate(); err != nil {
		t.Fatalf("atom activation policy: %v", err)
	}

	if got := policy.ProtocolAtHeight(99); got != ProtocolLegacyV1 {
		t.Fatalf(
			"height 99=%q want=%q",
			got,
			ProtocolLegacyV1,
		)
	}

	if got := policy.ProtocolAtHeight(100); got != ProtocolAtomV3 {
		t.Fatalf(
			"height 100=%q want=%q",
			got,
			ProtocolAtomV3,
		)
	}
}

func TestActivationPlanRejectsUnknownTargetProtocolV1(t *testing.T) {
	plan := atomActivationTestPlanV1()
	plan.TargetProtocol = ProtocolVersion("QC_UNKNOWN_PROTOCOL")

	if err := plan.Validate(); err == nil {
		t.Fatal("unknown target protocol unexpectedly accepted")
	}
}

func TestCanonicalActivationRemainsDisabledV1(t *testing.T) {
	policy := CanonicalActivationPolicyV1()

	if policy.Enabled {
		t.Fatal("canonical activation unexpectedly enabled")
	}

	for _, height := range []int{0, 1, 100, 1_000_000} {
		if got := policy.ProtocolAtHeight(height); got != ProtocolLegacyV1 {
			t.Fatalf(
				"height %d protocol=%q want=%q",
				height,
				got,
				ProtocolLegacyV1,
			)
		}
	}
}

func TestAtomV3CanonicalBlockValidationRemainsFailClosedV1(t *testing.T) {
	_, err := ValidateCanonicalBlockV1(
		CanonicalBlockValidationInputV1{
			Protocol: ProtocolAtomV3,
		},
	)

	if err == nil {
		t.Fatal("AtomV3 block validation unexpectedly accepted")
	}
}

func TestAtomV3ParamsRemainFailClosedUntilImplementedV1(t *testing.T) {
	params := LegacyParamsV1()
	params.Protocol = ProtocolAtomV3

	if err := params.Validate(); err == nil {
		t.Fatal("AtomV3 params unexpectedly accepted before implementation")
	}
}
