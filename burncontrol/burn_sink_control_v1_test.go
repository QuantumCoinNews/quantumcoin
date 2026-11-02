package burncontrol

import "testing"

func TestDefaultBurnSinkPolicyValid(t *testing.T) {
	policy := DefaultPolicyV1()

	if err := ValidatePolicyV1(policy); err != nil {
		t.Fatalf("default burn policy should validate: %v", err)
	}
}

func TestBurnSinkMaxSupplyInvariant(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.MaxSupplyQC != 25500000 {
		t.Fatalf("burn policy must not change max supply")
	}
	if !policy.BurnDoesNotReduceMaxSupply {
		t.Fatalf("burn must not reduce max supply constant")
	}
}

func TestBurnSinkRequiresUnspendableAddress(t *testing.T) {
	policy := DefaultPolicyV1()

	if !policy.BurnAddressMustBeUnspendable {
		t.Fatalf("burn address must be unspendable")
	}
	if !policy.BurnPrivateKeyMustNotExist {
		t.Fatalf("burn private key must not exist")
	}
}

func TestBurnSinkRejectsMintReplacement(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.BurnDoesNotMintReplacement = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("burn replacement mint must fail")
	}
}

func TestBurnSinkRejectsAutoBurn(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.AutoBurnForbidden = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("auto burn allowed must fail")
	}
}

func TestBurnSinkRejectsSpendableBurnAddress(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.BurnAddressMustBeUnspendable = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("spendable burn address must fail")
	}
}
