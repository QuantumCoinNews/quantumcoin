package node

import (
	"testing"

	"quantumcoin/backuprestore"
	"quantumcoin/burncontrol"
	"quantumcoin/economycontrol"
	"quantumcoin/foundervesting"
	"quantumcoin/hardforkpolicy"
	"quantumcoin/miningtransition"
	"quantumcoin/networkfee"
	"quantumcoin/persistence"
	"quantumcoin/queueworker"
)

func TestQMA0059SafeBoundaryRejectsUnexpectedKeysV1(t *testing.T) {
	t.Run("burn", func(t *testing.T) {
		p := burncontrol.DefaultPolicyV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := burncontrol.ValidatePolicyV1(p); err == nil {
			t.Fatal("burn policy accepted unexpected safe-boundary key")
		}
	})

	t.Run("economy", func(t *testing.T) {
		p := economycontrol.DefaultPolicyV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := economycontrol.ValidatePolicyV1(p); err == nil {
			t.Fatal("economy policy accepted unexpected safe-boundary key")
		}
	})

	t.Run("founder", func(t *testing.T) {
		p := foundervesting.DefaultPolicyV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := foundervesting.ValidatePolicyV1(p); err == nil {
			t.Fatal("founder policy accepted unexpected safe-boundary key")
		}
	})

	t.Run("hardfork", func(t *testing.T) {
		p := hardforkpolicy.DefaultPolicyV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := hardforkpolicy.ValidatePolicyV1(p); err == nil {
			t.Fatal("hardfork policy accepted unexpected safe-boundary key")
		}
	})

	t.Run("miningtransition", func(t *testing.T) {
		p := miningtransition.DefaultPolicyV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := miningtransition.ValidatePolicyV1(p); err == nil {
			t.Fatal("mining-transition policy accepted unexpected safe-boundary key")
		}
	})

	t.Run("networkfee", func(t *testing.T) {
		p := networkfee.DefaultPolicyV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := networkfee.ValidatePolicyV1(p); err == nil {
			t.Fatal("network-fee policy accepted unexpected safe-boundary key")
		}
	})

	t.Run("backuprestore", func(t *testing.T) {
		p := backuprestore.DefaultRunbookV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := backuprestore.ValidateRunbookV1(p); err == nil {
			t.Fatal("backup/restore runbook accepted unexpected safe-boundary key")
		}
	})

	t.Run("persistence", func(t *testing.T) {
		p := persistence.DefaultPlanV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := persistence.ValidatePlanV1(p); err == nil {
			t.Fatal("persistence plan accepted unexpected safe-boundary key")
		}
	})

	t.Run("queueworker_plan_only", func(t *testing.T) {
		p := queueworker.DefaultPlanV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := queueworker.ValidatePlanV1(p); err == nil {
			t.Fatal("queue-worker plan accepted unexpected safe-boundary key")
		}
	})

	t.Run("queueworker_runtime_manual", func(t *testing.T) {
		p := queueworker.RuntimePlanV1()
		p.SafeBoundary["unexpectedQMA0059Key"] = true
		if err := queueworker.ValidatePlanV1(p); err == nil {
			t.Fatal("runtime queue-worker plan accepted unexpected safe-boundary key")
		}
	})
}

func TestQMA0059SafeBoundaryRejectsDeletedCanonicalKeysV1(t *testing.T) {
	t.Run("burn", func(t *testing.T) {
		p := burncontrol.DefaultPolicyV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := burncontrol.ValidatePolicyV1(p); err == nil {
			t.Fatal("burn policy accepted deleted canonical key")
		}
	})

	t.Run("economy", func(t *testing.T) {
		p := economycontrol.DefaultPolicyV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := economycontrol.ValidatePolicyV1(p); err == nil {
			t.Fatal("economy policy accepted deleted canonical key")
		}
	})

	t.Run("founder", func(t *testing.T) {
		p := foundervesting.DefaultPolicyV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := foundervesting.ValidatePolicyV1(p); err == nil {
			t.Fatal("founder policy accepted deleted canonical key")
		}
	})

	t.Run("hardfork", func(t *testing.T) {
		p := hardforkpolicy.DefaultPolicyV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := hardforkpolicy.ValidatePolicyV1(p); err == nil {
			t.Fatal("hardfork policy accepted deleted canonical key")
		}
	})

	t.Run("miningtransition", func(t *testing.T) {
		p := miningtransition.DefaultPolicyV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := miningtransition.ValidatePolicyV1(p); err == nil {
			t.Fatal("mining-transition policy accepted deleted canonical key")
		}
	})

	t.Run("networkfee", func(t *testing.T) {
		p := networkfee.DefaultPolicyV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := networkfee.ValidatePolicyV1(p); err == nil {
			t.Fatal("network-fee policy accepted deleted canonical key")
		}
	})

	t.Run("backuprestore", func(t *testing.T) {
		p := backuprestore.DefaultRunbookV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := backuprestore.ValidateRunbookV1(p); err == nil {
			t.Fatal("backup/restore runbook accepted deleted canonical key")
		}
	})

	t.Run("persistence", func(t *testing.T) {
		p := persistence.DefaultPlanV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := persistence.ValidatePlanV1(p); err == nil {
			t.Fatal("persistence plan accepted deleted canonical key")
		}
	})

	t.Run("queueworker_plan_only", func(t *testing.T) {
		p := queueworker.DefaultPlanV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := queueworker.ValidatePlanV1(p); err == nil {
			t.Fatal("queue-worker plan accepted deleted canonical key")
		}
	})

	t.Run("queueworker_runtime_manual", func(t *testing.T) {
		p := queueworker.RuntimePlanV1()
		delete(p.SafeBoundary, "doesNotMint")
		if err := queueworker.ValidatePlanV1(p); err == nil {
			t.Fatal("runtime queue-worker plan accepted deleted canonical key")
		}
	})
}

func TestQMA0059QueueWorkerRuntimeBoundaryHasExactModeSchemaV1(t *testing.T) {
	plan := queueworker.RuntimePlanV1()

	if plan.SafeBoundary["doesNotStartWorkers"] {
		t.Fatal("runtime-manual plan must allow explicit worker start")
	}

	if plan.SafeBoundary["planOnly"] {
		t.Fatal("runtime-manual plan must not claim plan-only mode")
	}

	if err := queueworker.ValidatePlanV1(plan); err != nil {
		t.Fatalf(
			"canonical runtime-manual queue plan rejected: %v",
			err,
		)
	}
}
