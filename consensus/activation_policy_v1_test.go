package consensus

import "testing"

const activationPolicyTestGenesisHexV1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestActivationPolicyV1ZeroValueIsDisabledV1(
	t *testing.T,
) {
	var policy ActivationPolicyV1

	if err := policy.Validate(); err != nil {
		t.Fatalf(
			"zero-value disabled policy rejected: %v",
			err,
		)
	}

	if policy.Enabled {
		t.Fatal("zero-value policy must be disabled")
	}

	if got := policy.ProtocolAtHeight(1_000_000); got != ProtocolLegacyV1 {
		t.Fatalf(
			"disabled policy protocol = %q, want %q",
			got,
			ProtocolLegacyV1,
		)
	}
}

func TestActivationPolicyV1EnabledUsesValidatedPlanV1(
	t *testing.T,
) {
	policy := ActivationPolicyV1{
		Enabled: true,
		Plan: ActivationPlan{
			NetworkID:        "quantumcoin-test-policy-v1",
			LegacyGenesisHex: activationPolicyTestGenesisHexV1,
			ActivationHeight: 100,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}

	policy = verifiedActivationFixtureV1(t, policy)
	if err := policy.Validate(); err != nil {
		t.Fatalf("enabled activation policy rejected: %v", err)
	}

	if got := policy.ProtocolAtHeight(99); got != ProtocolLegacyV1 {
		t.Fatalf(
			"pre-activation protocol = %q, want %q",
			got,
			ProtocolLegacyV1,
		)
	}

	if got := policy.ProtocolAtHeight(100); got != ProtocolContinuityV2 {
		t.Fatalf(
			"activation protocol = %q, want %q",
			got,
			ProtocolContinuityV2,
		)
	}
}

func TestActivationPolicyV1RejectsIncompleteEnabledPlanV1(
	t *testing.T,
) {
	policy := ActivationPolicyV1{
		Enabled: true,
	}

	if err := policy.Validate(); err == nil {
		t.Fatal("enabled policy accepted incomplete plan")
	}
}

func TestActivationPolicyV1RejectsHiddenDisabledPlanV1(
	t *testing.T,
) {
	policy := ActivationPolicyV1{
		Plan: ActivationPlan{
			NetworkID:        "hidden-plan",
			LegacyGenesisHex: activationPolicyTestGenesisHexV1,
			ActivationHeight: 100,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}

	if err := policy.Validate(); err == nil {
		t.Fatal("disabled policy accepted hidden activation plan")
	}
}
