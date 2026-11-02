package economycontrol

import "testing"

func TestValidatePolicyV1RejectsAllocationBasisPointSwap(t *testing.T) {
	policy := DefaultPolicyV1()

	policy.Allocations[0].BasisPoints, policy.Allocations[2].BasisPoints =
		policy.Allocations[2].BasisPoints, policy.Allocations[0].BasisPoints

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatal("allocation basis-point swap must be rejected")
	}
}

func TestValidatePolicyV1RejectsFounderLiquidSemanticFlags(t *testing.T) {
	policy := DefaultPolicyV1()

	policy.Allocations[0].Locked = true
	policy.Allocations[0].Liquid = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatal("founder liquid allocation with wrong Locked/Liquid semantics must be rejected")
	}
}

func TestValidatePolicyV1RejectsFounderLockedSemanticFlags(t *testing.T) {
	policy := DefaultPolicyV1()

	policy.Allocations[1].Locked = false
	policy.Allocations[1].Liquid = true

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatal("founder locked allocation with wrong Locked/Liquid semantics must be rejected")
	}
}

func TestValidatePolicyV1RejectsNetworkSemanticFlags(t *testing.T) {
	policy := DefaultPolicyV1()

	policy.Allocations[2].Liquid = true

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatal("network allocation with wrong Locked/Liquid semantics must be rejected")
	}
}
