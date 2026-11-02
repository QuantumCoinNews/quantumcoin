package economycontrol

import "testing"

func TestDefaultMainnetBeforeEconomyPolicyValid(t *testing.T) {
	policy := DefaultPolicyV1()

	if err := ValidatePolicyV1(policy); err != nil {
		t.Fatalf("default economy policy should validate: %v", err)
	}
}

func TestEconomySupplyInvariant(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.TotalSupplyQC != 25500000 {
		t.Fatalf("total supply must stay 25,500,000 QC")
	}
}

func TestFounderSplitInvariant(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.FounderTotalBasisPoints != 500 {
		t.Fatalf("founder total must be 5%%")
	}
	if policy.FounderLiquidBasisPoints != 100 {
		t.Fatalf("founder liquid must be 1%%")
	}
	if policy.FounderLockedBasisPoints != 400 {
		t.Fatalf("founder locked must be 4%%")
	}
	if policy.FounderLiquidBasisPoints+policy.FounderLockedBasisPoints != policy.FounderTotalBasisPoints {
		t.Fatalf("founder liquid + locked must equal total")
	}
}

func TestFounderWalletBindingRequired(t *testing.T) {
	policy := DefaultPolicyV1()

	if !policy.FounderWalletBindingRequired {
		t.Fatalf("founder wallet binding must be required")
	}
	if !policy.FounderWalletAddressFinalBeforeMainnet {
		t.Fatalf("founder wallet address must be finalized before mainnet")
	}
	if !policy.SafeBoundary["requiresFounderWalletBinding"] {
		t.Fatalf("safe boundary must require founder wallet binding")
	}
	if !policy.SafeBoundary["requiresFinalFounderWalletAddress"] {
		t.Fatalf("safe boundary must require final founder wallet address")
	}
}

func TestEconomyRejectsSupplyChange(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.TotalSupplyQC = 25500001

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("supply change must fail")
	}
}

func TestEconomyRejectsMintingAllocation(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.Allocations[0].MintsNewQC = true

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("minting allocation must fail")
	}
}

func TestEconomyRejectsMissingSafeBoundary(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.SafeBoundary["doesNotChangeSupply"] = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("missing supply safe boundary must fail")
	}
}

func TestEconomyRejectsMissingFounderWalletBinding(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.FounderWalletBindingRequired = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("missing founder wallet binding requirement must fail")
	}
}
