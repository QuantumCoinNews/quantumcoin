package queueworker

import (
	"context"
	"errors"
	"testing"
)

func TestQueueValidatorOwnedPolicyCannotChangeAfterConstruction(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	queue, err := NewQueue(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Plan.Queues[0].RequiresAdminApproval = false
	cfg.Plan.Queues[0].AutoExecAllowed = true
	cfg.Plan.SafeBoundary["doesNotSendFunds"] = false
	job, _, err := queue.Enqueue(EnqueueRequest{ID: "protected", Queue: "tgwt_withdraw_review_queue", Payload: []byte(`{}`), IdempotencyKey: "protected"})
	if err != nil {
		t.Fatal(err)
	}
	if !job.RequiresApproval {
		t.Fatal("caller mutation removed approval")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := queue.Claim(ctx, ClaimRequest{Queue: job.Queue, Worker: "test"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("unapproved job claim: %v", err)
	}
	if !queue.config.Plan.SafeBoundary["doesNotSendFunds"] {
		t.Fatal("caller mutation changed owned boundary")
	}
}

func TestQueueValidatorApprovalBypass(t *testing.T) {
	for _, runtime := range []bool{false, true} {
		for _, kind := range []QueueKind{QueueKindReview, QueueKindSettlement, QueueKindPublishing, QueueKindMaintenance} {
			for _, auto := range []bool{false, true} {
				plan := DefaultPlanV1()
				if runtime {
					plan = RuntimePlanV1()
				}
				policy := newQueuePolicyV1("additional_review", kind, false, auto)
				plan.Queues = append(plan.Queues, policy)
				if err := ValidatePlanV1(plan); err == nil {
					t.Errorf("unapproved non-monitoring policy accepted: runtime=%v kind=%s auto=%v", runtime, kind, auto)
				}
			}
		}
	}
}

func TestQueueValidatorRequiredQueueCannotBeRelabeled(t *testing.T) {
	for index := range DefaultPlanV1().Queues {
		plan := RuntimePlanV1()
		policy := &plan.Queues[index]
		if policy.Kind == QueueKindMonitoring {
			policy.Kind = QueueKindReview
			policy.RequiresAdminApproval = true
			policy.AutoExecAllowed = false
		} else {
			policy.Kind = QueueKindMonitoring
			policy.RequiresAdminApproval = false
			policy.AutoExecAllowed = true
		}
		if err := ValidatePlanV1(plan); err == nil {
			t.Errorf("required queue kind changed: %s", policy.Name)
		}
		cfg := DefaultRuntimeConfig()
		cfg.Plan = plan
		if _, err := NewQueue(cfg); err == nil {
			t.Errorf("constructor bypass: %s", policy.Name)
		}
	}
}

func TestQueueValidatorControlsBothModes(t *testing.T) {
	for _, runtime := range []bool{false, true} {
		makePlan := func() Plan {
			if runtime {
				return RuntimePlanV1()
			}
			return DefaultPlanV1()
		}
		for _, kind := range []QueueKind{QueueKindReview, QueueKindSettlement, QueueKindPublishing, QueueKindMaintenance} {
			plan := makePlan()
			plan.Queues = append(plan.Queues, newQueuePolicyV1("extra_protected", kind, true, false))
			if err := ValidatePlanV1(plan); err != nil {
				t.Fatalf("valid protected queue: %v", err)
			}
		}
		cases := map[string]func(*Plan){
			"unknown_kind":        func(p *Plan) { p.Queues[0].Kind = "typo" },
			"missing_queue":       func(p *Plan) { p.Queues = p.Queues[:len(p.Queues)-1] },
			"unsafe_mint":         func(p *Plan) { p.Queues[0].CanMint = true },
			"unsafe_send":         func(p *Plan) { p.Queues[0].CanSendFunds = true },
			"unsafe_balance":      func(p *Plan) { p.Queues[0].CanChangeBalances = true },
			"unsafe_chain":        func(p *Plan) { p.Queues[0].CanChangeChain = true },
			"missing_audit":       func(p *Plan) { p.Queues[0].RequiresAuditLog = false },
			"missing_idempotency": func(p *Plan) { p.Queues[0].RequiresIdempotency = false },
			"missing_deadletter":  func(p *Plan) { p.Queues[0].RequiresDeadLetter = false },
			"testnet_mix":         func(p *Plan) { p.Queues[0].TestnetSeparated = false },
			"mainnet_mix":         func(p *Plan) { p.Queues[0].MainnetSeparated = false },
			"boundary_added":      func(p *Plan) { p.SafeBoundary["unrecognized"] = true },
			"boundary_removed":    func(p *Plan) { delete(p.SafeBoundary, "doesNotMint") },
			"unknown_mode":        func(p *Plan) { p.Mode = "unrecognized" },
			"unknown_version":     func(p *Plan) { p.Version = "unknown" },
			"duplicate":           func(p *Plan) { p.Queues = append(p.Queues, p.Queues[0]) },
		}
		for name, mutate := range cases {
			t.Run(name, func(t *testing.T) {
				p := makePlan()
				mutate(&p)
				if err := ValidatePlanV1(p); err == nil {
					t.Fatal("invalid plan accepted")
				}
				cfg := DefaultRuntimeConfig()
				cfg.Plan = p
				if _, err := NewQueue(cfg); err == nil {
					t.Fatal("invalid plan reached queue")
				}
			})
		}
	}
}
