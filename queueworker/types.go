package queueworker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusLeased     JobStatus = "leased"
	JobStatusSucceeded  JobStatus = "succeeded"
	JobStatusDeadLetter JobStatus = "dead_letter"
)

type AuditAction string

const (
	AuditActionEnqueued       AuditAction = "enqueued"
	AuditActionDuplicate      AuditAction = "duplicate"
	AuditActionApproved       AuditAction = "approved"
	AuditActionClaimed        AuditAction = "claimed"
	AuditActionLeaseExpired   AuditAction = "lease_expired"
	AuditActionLeaseRenewed   AuditAction = "lease_renewed"
	AuditActionAcknowledged   AuditAction = "acknowledged"
	AuditActionRetryScheduled AuditAction = "retry_scheduled"
	AuditActionDeadLettered   AuditAction = "dead_lettered"
	AuditActionShutdown       AuditAction = "shutdown"
)

var (
	ErrQueueClosed = errors.New("queue is shut down")
	ErrInvalidJob  = errors.New("invalid queue job")
	ErrJobNotFound = errors.New("queue job not found")
	ErrNotApproved = errors.New("queue job requires admin approval")
	ErrLeaseLost   = errors.New("queue job lease is not owned")
)

type Job struct {
	ID             string          `json:"id"`
	Queue          string          `json:"queue"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Status         JobStatus       `json:"status"`

	RequiresApproval bool   `json:"requiresApproval"`
	ApprovedBy       string `json:"approvedBy,omitempty"`
	ApprovedAt       int64  `json:"approvedAt,omitempty"`

	Attempts    int   `json:"attempts"`
	MaxAttempts int   `json:"maxAttempts"`
	AvailableAt int64 `json:"availableAt"`

	LeaseOwner string `json:"leaseOwner,omitempty"`
	LeaseUntil int64  `json:"leaseUntil,omitempty"`

	LastError string `json:"lastError,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type EnqueueRequest struct {
	ID             string
	Queue          string
	Payload        json.RawMessage
	IdempotencyKey string
	MaxAttempts    int
}

type ClaimRequest struct {
	Queue  string
	Worker string
}

type AuditEvent struct {
	Sequence uint64      `json:"sequence"`
	JobID    string      `json:"jobId,omitempty"`
	Queue    string      `json:"queue,omitempty"`
	Action   AuditAction `json:"action"`
	Actor    string      `json:"actor,omitempty"`
	Message  string      `json:"message,omitempty"`
	Time     int64       `json:"time"`
}

type Handler func(
	ctx context.Context,
	job Job,
) error

type RuntimeConfig struct {
	Plan          Plan
	LeaseDuration time.Duration
	BackoffBase   time.Duration
	MaxAttempts   int
	AuditLimit    int
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Plan:          RuntimePlanV1(),
		LeaseDuration: 30 * time.Second,
		BackoffBase:   time.Second,
		MaxAttempts:   5,
		AuditLimit:    10_000,
	}
}

func (cfg RuntimeConfig) Validate() error {
	if err := ValidatePlanV1(cfg.Plan); err != nil {
		return fmt.Errorf("validate queue policy plan: %w", err)
	}

	if cfg.LeaseDuration <= 0 {
		return errors.New("queue lease duration must be positive")
	}

	if cfg.BackoffBase <= 0 {
		return errors.New("queue backoff base must be positive")
	}

	if cfg.MaxAttempts < 1 {
		return errors.New(
			"queue maximum attempts must be at least one",
		)
	}

	if cfg.AuditLimit < 1 {
		return errors.New(
			"queue audit limit must be at least one",
		)
	}

	return nil
}

func (request EnqueueRequest) Validate() error {
	if strings.TrimSpace(request.ID) == "" {
		return fmt.Errorf("%w: job ID required", ErrInvalidJob)
	}

	if request.ID != strings.TrimSpace(request.ID) {
		return fmt.Errorf(
			"%w: job ID contains surrounding whitespace",
			ErrInvalidJob,
		)
	}

	if strings.TrimSpace(request.Queue) == "" {
		return fmt.Errorf(
			"%w: queue name required",
			ErrInvalidJob,
		)
	}

	if request.Queue != strings.TrimSpace(request.Queue) {
		return fmt.Errorf(
			"%w: queue name contains surrounding whitespace",
			ErrInvalidJob,
		)
	}

	if strings.TrimSpace(request.IdempotencyKey) == "" {
		return fmt.Errorf(
			"%w: idempotency key required",
			ErrInvalidJob,
		)
	}

	if request.IdempotencyKey !=
		strings.TrimSpace(request.IdempotencyKey) {
		return fmt.Errorf(
			"%w: idempotency key contains surrounding whitespace",
			ErrInvalidJob,
		)
	}

	if len(request.Payload) == 0 {
		return fmt.Errorf(
			"%w: payload required",
			ErrInvalidJob,
		)
	}

	if !json.Valid(request.Payload) {
		return fmt.Errorf(
			"%w: payload must contain valid JSON",
			ErrInvalidJob,
		)
	}

	if request.MaxAttempts < 0 {
		return fmt.Errorf(
			"%w: maximum attempts cannot be negative",
			ErrInvalidJob,
		)
	}

	return nil
}

func (request ClaimRequest) Validate() error {
	if strings.TrimSpace(request.Queue) == "" {
		return errors.New("claim queue name required")
	}

	if request.Queue != strings.TrimSpace(request.Queue) {
		return errors.New(
			"claim queue name contains surrounding whitespace",
		)
	}

	if strings.TrimSpace(request.Worker) == "" {
		return errors.New("claim worker identity required")
	}

	if request.Worker != strings.TrimSpace(request.Worker) {
		return errors.New(
			"claim worker identity contains surrounding whitespace",
		)
	}

	return nil
}

func IsKnownJobStatus(status JobStatus) bool {
	switch status {
	case JobStatusPending,
		JobStatusLeased,
		JobStatusSucceeded,
		JobStatusDeadLetter:
		return true
	default:
		return false
	}
}

func queuePolicyByName(
	plan Plan,
	name string,
) (QueuePolicy, bool) {
	for _, policy := range plan.Queues {
		if policy.Name == name {
			return policy, true
		}
	}

	return QueuePolicy{}, false
}

func cloneJob(job *Job) Job {
	if job == nil {
		return Job{}
	}

	result := *job
	result.Payload = append(
		json.RawMessage(nil),
		job.Payload...,
	)

	return result
}

func cloneAuditEvent(event AuditEvent) AuditEvent {
	return event
}
