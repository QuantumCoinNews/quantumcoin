package hardforkpolicy

import "testing"

func TestDefaultHardforkPolicyValid(t *testing.T) {
	policy := DefaultPolicyV1()

	if err := ValidatePolicyV1(policy); err != nil {
		t.Fatalf("default hardfork policy should validate: %v", err)
	}
}

func TestHardforkDoesNotExecuteNow(t *testing.T) {
	policy := DefaultPolicyV1()

	if policy.HardforkExecutionNow {
		t.Fatalf("policy must not execute hardfork now")
	}
}

func TestHardforkEmergencyAllowed(t *testing.T) {
	policy := DefaultPolicyV1()

	if !policy.HardforkAllowed {
		t.Fatalf("hardfork must be available for emergencies")
	}
	if len(policy.AllowedReasons) < 6 {
		t.Fatalf("allowed reasons missing")
	}
}

func TestHardforkForbiddenEconomyChanges(t *testing.T) {
	policy := DefaultPolicyV1()

	if !policy.HardforkCannotChangeMaxSupply {
		t.Fatalf("hardfork must not change max supply")
	}
	if !policy.HardforkCannotMintExtraQC {
		t.Fatalf("hardfork must not mint extra QC")
	}
	if !policy.HardforkCannotBypassFounderVesting {
		t.Fatalf("hardfork must not bypass founder vesting")
	}
	if !policy.HardforkCannotBypassBurnRules {
		t.Fatalf("hardfork must not bypass burn rules")
	}
}

func TestHardforkRequiresProcess(t *testing.T) {
	policy := DefaultPolicyV1()

	if !policy.RequiresTestnetRehearsal {
		t.Fatalf("hardfork must require testnet rehearsal")
	}
	if !policy.RequiresMigrationHeight {
		t.Fatalf("hardfork must require migration height")
	}
	if !policy.RequiresRollbackPlan {
		t.Fatalf("hardfork must require rollback plan")
	}
}

func TestRejectHardforkExecutionNow(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.HardforkExecutionNow = true

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("hardfork execution now must fail")
	}
}

func TestRejectMaxSupplyChangePermission(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.HardforkCannotChangeMaxSupply = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("max supply change permission must fail")
	}
}

func TestRejectAIHardforkPermission(t *testing.T) {
	policy := DefaultPolicyV1()
	policy.HardforkCannotAutoRunByAI = false

	if err := ValidatePolicyV1(policy); err == nil {
		t.Fatalf("AI hardfork permission must fail")
	}
}
