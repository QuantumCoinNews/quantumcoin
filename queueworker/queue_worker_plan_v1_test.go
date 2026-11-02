package queueworker

import "testing"

func TestDefaultQueueWorkerPlanValid(t *testing.T) {
	plan := DefaultPlanV1()

	if err := ValidatePlanV1(plan); err != nil {
		t.Fatalf("default queue worker plan should validate: %v", err)
	}
	if err := HasRequiredQueuesV1(plan); err != nil {
		t.Fatalf("default queue worker plan missing queue: %v", err)
	}
}

func TestQueueWorkerSafeBoundary(t *testing.T) {
	plan := DefaultPlanV1()
	for _, k := range []string{
		"doesNotStartWorkers",
		"doesNotMint",
		"doesNotSendFunds",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeSupply",
		"requiresIdempotency",
		"requiresDeadLetter",
		"requiresAuditLog",
	} {
		if !plan.SafeBoundary[k] {
			t.Fatalf("safe boundary missing: %s", k)
		}
	}
}

func TestQueueWorkerRejectsUnsafeCapability(t *testing.T) {
	plan := DefaultPlanV1()
	plan.Queues[0].CanSendFunds = true

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatalf("queue with send funds capability must fail")
	}
}

func TestQueueWorkerRejectsMissingDeadLetter(t *testing.T) {
	plan := DefaultPlanV1()
	plan.Queues[0].RequiresDeadLetter = false

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatalf("queue without dead-letter policy must fail")
	}
}

func TestQueueWorkerRejectsDuplicateQueue(t *testing.T) {
	plan := DefaultPlanV1()
	plan.Queues = append(plan.Queues, plan.Queues[0])

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatalf("duplicate queue must fail")
	}
}
func TestQueueWorkerRejectsUnknownKindV1(t *testing.T) {
	plan := DefaultPlanV1()
	plan.Queues[0].Kind = QueueKind("unknown_kind")

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatalf("unknown queue kind must fail")
	}
}

func TestQueueWorkerRejectsAutoExecForNonMonitoringV1(t *testing.T) {
	plan := DefaultPlanV1()
	plan.Queues[0].AutoExecAllowed = true

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatalf("non-monitoring auto-execution must fail")
	}
}

func TestQueueWorkerRejectsAutoExecWithAdminApprovalV1(t *testing.T) {
	plan := DefaultPlanV1()
	plan.Queues[3].AutoExecAllowed = true
	plan.Queues[3].RequiresAdminApproval = true

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatalf("auto-execution with admin approval must fail")
	}
}

func TestQueueWorkerRequiresAllQueuesInsideValidatorV1(t *testing.T) {
	plan := DefaultPlanV1()
	plan.Queues = plan.Queues[:len(plan.Queues)-1]

	if err := ValidatePlanV1(plan); err == nil {
		t.Fatalf("missing required queue must fail inside validator")
	}
}
