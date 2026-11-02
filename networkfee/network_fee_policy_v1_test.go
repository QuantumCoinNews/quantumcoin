package networkfee

import "testing"

func TestDefaultNetworkFeePolicyValid(t *testing.T) {
	policy := DefaultPolicyV1()

	if err := ValidatePolicyV1(policy); err != nil {
		t.Fatalf("default network fee policy should validate: %v", err)
	}
}

func TestNetworkFeeMicroLow(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.BaseFeeTargetQC != "0.000001" {
		t.Fatalf("base fee target must be 0.000001 QC")
	}
	if policy.CongestionCeilingQC != "0.00001" {
		t.Fatalf("congestion ceiling must be 0.00001 QC")
	}
	if !policy.FeeMustStayMicroLow {
		t.Fatalf("fee must stay micro-low")
	}
}

func TestNetworkFeeDestination(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.FeeRecipient != "BLOCK_PRODUCER_MINER" {
		t.Fatalf("fee recipient must be block producer/miner")
	}
	if policy.FeeGoesToFounder {
		t.Fatalf("fee must not go to founder")
	}
	if policy.FeeGoesToBurnByDefault {
		t.Fatalf("fee must not burn by default")
	}
}

func TestNetworkFeeNotMint(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.FeeIsNewMint {
		t.Fatalf("fee must not be new mint")
	}
	if policy.FeeChangesMaxSupply {
		t.Fatalf("fee must not change max supply")
	}
}

func TestRejectOldHigherBaseFee(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.BaseFeeTargetQC = "0.0001"

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("old higher base fee must fail")
	}
}

func TestRejectFounderFee(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.FeeGoesToFounder = true

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("founder fee destination must fail")
	}
}

func TestRejectAutoFeeChange(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.AICannotChangeFee = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("AI fee change permission must fail")
	}
}
