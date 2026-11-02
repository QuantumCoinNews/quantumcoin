package consensus

import "testing"

func TestCanonicalActivationPolicyV1IsSafelyDisabledV1(
	t *testing.T,
) {
	policy := CanonicalActivationPolicyV1()

	if err := policy.Validate(); err != nil {
		t.Fatalf(
			"canonical activation policy rejected: %v",
			err,
		)
	}

	if policy.Enabled {
		t.Fatal(
			"canonical activation policy must remain disabled until configured",
		)
	}

	if policy.Plan != (ActivationPlan{}) {
		t.Fatal(
			"disabled canonical policy must not contain a hidden plan",
		)
	}

	for _, height := range []int{
		-1,
		0,
		1,
		1_000_000,
	} {
		if got := policy.ProtocolAtHeight(height); got != ProtocolLegacyV1 {
			t.Fatalf(
				"height %d protocol = %q, want %q",
				height,
				got,
				ProtocolLegacyV1,
			)
		}
	}
}

func TestCanonicalActivationPolicyV1ReturnsIndependentValueV1(
	t *testing.T,
) {
	first := CanonicalActivationPolicyV1()
	first.Enabled = true
	first.Plan.NetworkID = "mutated-locally"

	second := CanonicalActivationPolicyV1()

	if second.Enabled {
		t.Fatal(
			"local mutation changed canonical enabled state",
		)
	}

	if second.Plan != (ActivationPlan{}) {
		t.Fatal(
			"local mutation changed canonical activation plan",
		)
	}
}
