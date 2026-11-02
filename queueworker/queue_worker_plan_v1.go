// Package queueworker contains QuantumCoin project code for this module.
package queueworker

import (
	"errors"
	"fmt"
	"strings"
)

const QueueWorkerPlanVersionV1 = "QC_QUEUE_WORKER_INTEGRATION_PLAN_V1"

type QueueMode string

const (
	QueueModePlanOnly QueueMode = "PLAN_ONLY_NO_WORKER_START"

	QueueModeRuntimeManual QueueMode = "RUNTIME_MANUAL_START"
)

type QueueKind string

const (
	QueueKindReview      QueueKind = "review"
	QueueKindSettlement  QueueKind = "settlement"
	QueueKindMonitoring  QueueKind = "monitoring"
	QueueKindPublishing  QueueKind = "publishing"
	QueueKindMaintenance QueueKind = "maintenance"
)

type QueuePolicy struct {
	Name                  string    `json:"name"`
	Kind                  QueueKind `json:"kind"`
	TestnetSeparated      bool      `json:"testnetSeparated"`
	MainnetSeparated      bool      `json:"mainnetSeparated"`
	RequiresAdminApproval bool      `json:"requiresAdminApproval"`
	RequiresIdempotency   bool      `json:"requiresIdempotency"`
	RequiresDeadLetter    bool      `json:"requiresDeadLetter"`
	RequiresAuditLog      bool      `json:"requiresAuditLog"`
	AutoExecAllowed       bool      `json:"autoExecAllowed"`
	CanSendFunds          bool      `json:"canSendFunds"`
	CanMint               bool      `json:"canMint"`
	CanChangeBalances     bool      `json:"canChangeBalances"`
	CanChangeChain        bool      `json:"canChangeChain"`
}

type Plan struct {
	Version      string            `json:"version"`
	Mode         QueueMode         `json:"mode"`
	Queues       []QueuePolicy     `json:"queues"`
	SafeBoundary map[string]bool   `json:"safeBoundary"`
	Notes        map[string]string `json:"notes"`
}

func DefaultPlanV1() Plan {
	return Plan{
		Version:      QueueWorkerPlanVersionV1,
		Mode:         QueueModePlanOnly,
		Queues:       defaultQueuePoliciesV1(),
		SafeBoundary: SafeBoundaryV1(),
		Notes: map[string]string{
			"dev": "Queue worker integration defaults to " +
				"plan-only mode.",
			"testnet": "Testnet queue names, DB paths, jobs, " +
				"and workers must be separated from mainnet.",
			"mainnet": "Mainnet workers require explicit manual " +
				"activation, admin approval, backup/restore " +
				"readiness, idempotency, audit log, and tests.",
			"ai": "AI can advise, score, and report only. " +
				"AI cannot execute settlement, mint, send funds, " +
				"or alter balances.",
		},
	}
}

// RuntimePlanV1 returns an explicitly activated manual-start plan.
// Constructing this value does not start a worker.
func RuntimePlanV1() Plan {
	plan := DefaultPlanV1()
	plan.Mode = QueueModeRuntimeManual
	plan.SafeBoundary["doesNotStartWorkers"] = false
	plan.SafeBoundary["planOnly"] = false
	plan.Notes["runtime"] = "Workers may start only through an " +
		"explicit Start call and remain subject to queue policy."

	return plan
}

func defaultQueuePoliciesV1() []QueuePolicy {
	return []QueuePolicy{
		newQueuePolicyV1(
			"tgwt_withdraw_review_queue",
			QueueKindReview,
			true,
			false,
		),
		newQueuePolicyV1(
			"tgwt_bsc_claim_settlement_queue",
			QueueKindSettlement,
			true,
			false,
		),
		newQueuePolicyV1(
			"monthly_nft_review_queue",
			QueueKindReview,
			true,
			false,
		),
		newQueuePolicyV1(
			"monitoring_alert_queue",
			QueueKindMonitoring,
			false,
			true,
		),
		newQueuePolicyV1(
			"autopublisher_content_queue",
			QueueKindPublishing,
			true,
			false,
		),
		newQueuePolicyV1(
			"backup_restore_check_queue",
			QueueKindMaintenance,
			true,
			false,
		),
	}
}

func newQueuePolicyV1(
	name string,
	kind QueueKind,
	requiresAdminApproval bool,
	autoExecAllowed bool,
) QueuePolicy {
	return QueuePolicy{
		Name:                  name,
		Kind:                  kind,
		TestnetSeparated:      true,
		MainnetSeparated:      true,
		RequiresAdminApproval: requiresAdminApproval,
		RequiresIdempotency:   true,
		RequiresDeadLetter:    true,
		RequiresAuditLog:      true,
		AutoExecAllowed:       autoExecAllowed,
	}
}

func SafeBoundaryV1() map[string]bool {
	return map[string]bool{
		"doesNotStartWorkers":    true,
		"doesNotMint":            true,
		"doesNotSendFunds":       true,
		"doesNotChangeBalances":  true,
		"doesNotChangeChain":     true,
		"doesNotChangeConsensus": true,
		"doesNotChangeSupply":    true,
		"doesNotStartMining":     true,
		"doesNotStopMining":      true,
		"planOnly":               true,
		"requiresAdminApproval":  true,
		"requiresIdempotency":    true,
		"requiresDeadLetter":     true,
		"requiresAuditLog":       true,
	}
}

func IsSupportedQueueModeV1(mode QueueMode) bool {
	switch mode {
	case QueueModePlanOnly,
		QueueModeRuntimeManual:
		return true
	default:
		return false
	}
}

func IsSupportedQueueKindV1(kind QueueKind) bool {
	switch kind {
	case QueueKindReview,
		QueueKindSettlement,
		QueueKindMonitoring,
		QueueKindPublishing,
		QueueKindMaintenance:
		return true
	default:
		return false
	}
}

func ValidatePlanV1(plan Plan) error {
	if strings.TrimSpace(plan.Version) !=
		QueueWorkerPlanVersionV1 {
		return errors.New(
			"invalid queue worker plan version",
		)
	}

	if !IsSupportedQueueModeV1(plan.Mode) {
		return fmt.Errorf(
			"unsupported queue worker mode: %q",
			plan.Mode,
		)
	}

	if len(plan.Queues) == 0 {
		return errors.New("queue policies required")
	}

	seen := make(map[string]struct{}, len(plan.Queues))

	for index, policy := range plan.Queues {
		if err := validateQueuePolicyV1(
			policy,
			index,
		); err != nil {
			return err
		}

		if _, exists := seen[policy.Name]; exists {
			return errors.New(
				"duplicate queue: " + policy.Name,
			)
		}

		seen[policy.Name] = struct{}{}
	}

	if err := validateQueueSafeBoundaryV1(
		plan.Mode,
		plan.SafeBoundary,
	); err != nil {
		return err
	}

	if err := HasRequiredQueuesV1(plan); err != nil {
		return err
	}

	return nil
}

func validateQueuePolicyV1(
	policy QueuePolicy,
	index int,
) error {
	name := strings.TrimSpace(policy.Name)
	if name == "" {
		return fmt.Errorf(
			"queue name required at index %d",
			index,
		)
	}

	if name != policy.Name {
		return fmt.Errorf(
			"queue name contains surrounding whitespace: %q",
			policy.Name,
		)
	}

	if !IsSupportedQueueKindV1(policy.Kind) {
		return fmt.Errorf(
			"unsupported queue kind %q: %s",
			policy.Kind,
			name,
		)
	}

	if !policy.TestnetSeparated ||
		!policy.MainnetSeparated {
		return errors.New(
			"testnet/mainnet queue separation required: " +
				name,
		)
	}

	if !policy.RequiresIdempotency {
		return errors.New(
			"idempotency required: " + name,
		)
	}

	if !policy.RequiresDeadLetter {
		return errors.New(
			"dead-letter queue required: " + name,
		)
	}

	if !policy.RequiresAuditLog {
		return errors.New(
			"audit log required: " + name,
		)
	}

	if policy.CanSendFunds ||
		policy.CanMint ||
		policy.CanChangeBalances ||
		policy.CanChangeChain {
		return errors.New(
			"unsafe queue capability forbidden: " + name,
		)
	}

	if policy.RequiresAdminApproval &&
		policy.AutoExecAllowed {
		return errors.New(
			"approval-protected queue cannot auto-execute: " +
				name,
		)
	}

	return nil
}

func validateQueueSafeBoundaryV1(
	mode QueueMode,
	boundary map[string]bool,
) error {
	if boundary == nil {
		return errors.New(
			"queue worker safe boundary required",
		)
	}
	expectedBoundary := SafeBoundaryV1()

	switch mode {
	case QueueModePlanOnly:
	// Canonical plan-only boundary remains unchanged.

	case QueueModeRuntimeManual:
		expectedBoundary["doesNotStartWorkers"] = false
		expectedBoundary["planOnly"] = false

	default:
		return fmt.Errorf(
			"unsupported queue worker mode: %q",
			mode,
		)
	}

	if err := validateExactSafeBoundaryMapV1(
		boundary,
		expectedBoundary,
	); err != nil {
		return err
	}

	requiredTrue := []string{
		"doesNotMint",
		"doesNotSendFunds",
		"doesNotChangeBalances",
		"doesNotChangeChain",
		"doesNotChangeConsensus",
		"doesNotChangeSupply",
		"doesNotStartMining",
		"doesNotStopMining",
		"requiresAdminApproval",
		"requiresIdempotency",
		"requiresDeadLetter",
		"requiresAuditLog",
	}

	for _, key := range requiredTrue {
		if !boundary[key] {
			return errors.New(
				"safe boundary missing: " + key,
			)
		}
	}

	switch mode {
	case QueueModePlanOnly:
		if !boundary["doesNotStartWorkers"] ||
			!boundary["planOnly"] {
			return errors.New(
				"plan-only mode requires worker-start boundary",
			)
		}

	case QueueModeRuntimeManual:
		if boundary["doesNotStartWorkers"] ||
			boundary["planOnly"] {
			return errors.New(
				"runtime mode must not claim plan-only boundary",
			)
		}
	}

	return nil
}

func RequiredQueueNamesV1() []string {
	return []string{
		"tgwt_withdraw_review_queue",
		"tgwt_bsc_claim_settlement_queue",
		"monthly_nft_review_queue",
		"monitoring_alert_queue",
		"autopublisher_content_queue",
		"backup_restore_check_queue",
	}
}

func HasRequiredQueuesV1(plan Plan) error {
	seen := make(
		map[string]struct{},
		len(plan.Queues),
	)

	for _, policy := range plan.Queues {
		name := strings.TrimSpace(policy.Name)
		if name == "" {
			continue
		}

		seen[name] = struct{}{}
	}

	for _, name := range RequiredQueueNamesV1() {
		if _, exists := seen[name]; !exists {
			return errors.New(
				"missing required queue: " + name,
			)
		}
	}

	return nil
}

func validateExactSafeBoundaryMapV1(
	boundary map[string]bool,
	expected map[string]bool,
) error {
	if boundary == nil {
		return errors.New("queue worker plan safe boundary required")
	}

	if len(boundary) != len(expected) {
		return errors.New("queue worker plan safe boundary key count mismatch")
	}

	for key, want := range expected {
		got, exists := boundary[key]
		if !exists {
			return errors.New("safe boundary missing: " + key)
		}
		if got != want {
			return errors.New("safe boundary value mismatch: " + key)
		}
	}

	for key := range boundary {
		if _, exists := expected[key]; !exists {
			return errors.New("unexpected safe boundary key: " + key)
		}
	}

	return nil
}
