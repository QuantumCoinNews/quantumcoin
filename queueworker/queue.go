package queueworker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const maximumRetryBackoffV1 = time.Hour

type Queue struct {
	mu sync.Mutex

	config RuntimeConfig
	now    func() time.Time
	store  StateStore

	jobs        map[string]*Job
	order       []string
	idempotency map[string]string

	audit         []AuditEvent
	auditSequence uint64

	notify chan struct{}
	closed bool
}

func NewQueue(config RuntimeConfig) (*Queue, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Queue{
		config:      config,
		now:         time.Now,
		jobs:        make(map[string]*Job),
		idempotency: make(map[string]string),
		notify:      make(chan struct{}, 1),
	}, nil
}

// Enqueue adds one validated job. Reusing the same queue and idempotency key
// returns the original job without creating a second execution.
func (queue *Queue) Enqueue(
	request EnqueueRequest,
) (Job, bool, error) {
	if queue == nil {
		return Job{}, false, ErrQueueClosed
	}

	if err := request.Validate(); err != nil {
		return Job{}, false, err
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		return Job{}, false, ErrQueueClosed
	}

	policy, exists := queuePolicyByName(
		queue.config.Plan,
		request.Queue,
	)
	if !exists {
		return Job{}, false, fmt.Errorf(
			"%w: unknown queue %q",
			ErrInvalidJob,
			request.Queue,
		)
	}

	idempotencyKey := compoundIdempotencyKeyV1(
		request.Queue,
		request.IdempotencyKey,
	)

	if existingID, exists := queue.idempotency[idempotencyKey]; exists {
		existing := queue.jobs[existingID]
		if existing == nil {
			return Job{}, false, errors.New(
				"queue idempotency index is inconsistent",
			)
		}

		before := queue.snapshotLocked()
		result := cloneJob(existing)

		queue.appendAuditLocked(
			existing,
			AuditActionDuplicate,
			"",
			"duplicate enqueue returned existing job",
		)

		if err := queue.commitMutationLocked(before); err != nil {
			return Job{}, false, fmt.Errorf(
				"persist duplicate enqueue audit: %w",
				err,
			)
		}

		return result, false, nil
	}

	if _, exists := queue.jobs[request.ID]; exists {
		return Job{}, false, fmt.Errorf(
			"%w: duplicate job ID %q",
			ErrInvalidJob,
			request.ID,
		)
	}

	maxAttempts := request.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = queue.config.MaxAttempts
	}

	now := queue.now().UnixNano()

	job := &Job{
		ID:               request.ID,
		Queue:            request.Queue,
		Payload:          append([]byte(nil), request.Payload...),
		IdempotencyKey:   request.IdempotencyKey,
		Status:           JobStatusPending,
		RequiresApproval: policy.RequiresAdminApproval,
		MaxAttempts:      maxAttempts,
		AvailableAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	before := queue.snapshotLocked()

	queue.jobs[job.ID] = job
	queue.order = append(queue.order, job.ID)
	queue.idempotency[idempotencyKey] = job.ID

	queue.appendAuditLocked(
		job,
		AuditActionEnqueued,
		"",
		"job accepted into queue",
	)

	result := cloneJob(job)

	if err := queue.commitMutationLocked(before); err != nil {
		return Job{}, false, fmt.Errorf(
			"persist enqueue: %w",
			err,
		)
	}

	queue.signalLocked()

	return result, true, nil
}

// Approve records explicit administrative approval for a protected job.
func (queue *Queue) Approve(
	jobID string,
	admin string,
) (Job, error) {
	if queue == nil {
		return Job{}, ErrQueueClosed
	}

	jobID = strings.TrimSpace(jobID)
	admin = strings.TrimSpace(admin)

	if jobID == "" {
		return Job{}, errors.New("approval job ID required")
	}
	if admin == "" {
		return Job{}, errors.New("approval administrator required")
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		return Job{}, ErrQueueClosed
	}

	job := queue.jobs[jobID]
	if job == nil {
		return Job{}, ErrJobNotFound
	}

	if job.Status != JobStatusPending {
		return Job{}, fmt.Errorf(
			"job cannot be approved in status %q",
			job.Status,
		)
	}

	if !job.RequiresApproval {
		return Job{}, errors.New(
			"job does not require administrative approval",
		)
	}

	if job.ApprovedBy != "" {
		if job.ApprovedBy != admin {
			return Job{}, errors.New(
				"job was already approved by another administrator",
			)
		}

		return cloneJob(job), nil
	}

	before := queue.snapshotLocked()
	now := queue.now().UnixNano()

	job.ApprovedBy = admin
	job.ApprovedAt = now
	job.UpdatedAt = now

	queue.appendAuditLocked(
		job,
		AuditActionApproved,
		admin,
		"administrative approval recorded",
	)

	result := cloneJob(job)

	if err := queue.commitMutationLocked(before); err != nil {
		return Job{}, fmt.Errorf(
			"persist administrative approval: %w",
			err,
		)
	}

	queue.signalLocked()

	return result, nil
}

// Claim waits until an eligible job is available or ctx is cancelled.
// A successful claim owns the job only until LeaseUntil.
func (queue *Queue) Claim(
	ctx context.Context,
	request ClaimRequest,
) (Job, error) {
	if queue == nil {
		return Job{}, ErrQueueClosed
	}
	if ctx == nil {
		return Job{}, errors.New("claim context required")
	}
	if err := request.Validate(); err != nil {
		return Job{}, err
	}

	if _, exists := queuePolicyByName(
		queue.config.Plan,
		request.Queue,
	); !exists {
		return Job{}, fmt.Errorf(
			"unknown claim queue: %s",
			request.Queue,
		)
	}

	for {
		queue.mu.Lock()

		if queue.closed {
			queue.mu.Unlock()
			return Job{}, ErrQueueClosed
		}

		before := queue.snapshotLocked()
		beforeAuditSequence := queue.auditSequence

		now := queue.now()
		queue.recoverExpiredLeasesLocked(now)

		job := queue.claimableJobLocked(
			request.Queue,
			now,
		)
		if job != nil {
			job.Status = JobStatusLeased
			job.Attempts++
			job.LeaseOwner = request.Worker
			job.LeaseUntil = now.Add(
				queue.config.LeaseDuration,
			).UnixNano()
			job.UpdatedAt = now.UnixNano()

			queue.appendAuditLocked(
				job,
				AuditActionClaimed,
				request.Worker,
				fmt.Sprintf(
					"lease acquired for attempt %d",
					job.Attempts,
				),
			)

			result := cloneJob(job)

			if err := queue.commitMutationLocked(
				before,
			); err != nil {
				queue.mu.Unlock()

				return Job{}, fmt.Errorf(
					"persist claimed job: %w",
					err,
				)
			}

			queue.mu.Unlock()
			return result, nil
		}

		if queue.auditSequence != beforeAuditSequence {
			if err := queue.commitMutationLocked(
				before,
			); err != nil {
				queue.mu.Unlock()

				return Job{}, fmt.Errorf(
					"persist expired lease recovery: %w",
					err,
				)
			}
		}

		wait := queue.nextWakeLocked(
			request.Queue,
			now,
		)
		notify := queue.notify
		queue.mu.Unlock()

		if wait < 0 {
			select {
			case <-ctx.Done():
				return Job{}, ctx.Err()
			case <-notify:
				continue
			}
		}

		timer := time.NewTimer(wait)

		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return Job{}, ctx.Err()

		case <-notify:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			continue

		case <-timer.C:
			continue
		}
	}
}

// RenewLease extends a currently owned lease.
func (queue *Queue) RenewLease(
	jobID string,
	worker string,
) (Job, error) {
	if queue == nil {
		return Job{}, ErrQueueClosed
	}

	jobID = strings.TrimSpace(jobID)
	worker = strings.TrimSpace(worker)

	if jobID == "" || worker == "" {
		return Job{}, errors.New(
			"lease renewal job ID and worker required",
		)
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		return Job{}, ErrQueueClosed
	}

	before := queue.snapshotLocked()
	beforeAuditSequence := queue.auditSequence

	now := queue.now()
	queue.recoverExpiredLeasesLocked(now)

	job := queue.jobs[jobID]
	if job == nil {
		if queue.auditSequence != beforeAuditSequence {
			if err := queue.commitMutationLocked(
				before,
			); err != nil {
				return Job{}, fmt.Errorf(
					"persist lease recovery: %w",
					err,
				)
			}
		}

		return Job{}, ErrJobNotFound
	}

	if job.Status != JobStatusLeased ||
		job.LeaseOwner != worker {
		if queue.auditSequence != beforeAuditSequence {
			if err := queue.commitMutationLocked(
				before,
			); err != nil {
				return Job{}, fmt.Errorf(
					"persist lease recovery: %w",
					err,
				)
			}
		}

		return Job{}, ErrLeaseLost
	}

	job.LeaseUntil = now.Add(
		queue.config.LeaseDuration,
	).UnixNano()
	job.UpdatedAt = now.UnixNano()

	queue.appendAuditLocked(
		job,
		AuditActionLeaseRenewed,
		worker,
		"worker lease renewed",
	)

	result := cloneJob(job)

	if err := queue.commitMutationLocked(before); err != nil {
		return Job{}, fmt.Errorf(
			"persist renewed lease: %w",
			err,
		)
	}

	return result, nil
}

// Ack marks a leased job as successfully completed and persists the mutation.
func (queue *Queue) Ack(
	jobID string,
	worker string,
) (Job, error) {
	if queue == nil {
		return Job{}, ErrQueueClosed
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		return Job{}, ErrQueueClosed
	}

	before := queue.snapshotLocked()
	beforeAuditSequence := queue.auditSequence

	now := queue.now()
	queue.recoverExpiredLeasesLocked(now)

	job, err := queue.ownedLeaseLocked(jobID, worker)
	if err != nil {
		if queue.auditSequence != beforeAuditSequence {
			if persistErr := queue.commitMutationLocked(before); persistErr != nil {
				return Job{}, fmt.Errorf(
					"persist lease recovery: %w",
					persistErr,
				)
			}
		}
		return Job{}, err
	}

	job.Status = JobStatusSucceeded
	job.LeaseOwner = ""
	job.LeaseUntil = 0
	job.LastError = ""
	job.UpdatedAt = now.UnixNano()

	queue.appendAuditLocked(
		job,
		AuditActionAcknowledged,
		worker,
		"job completed successfully",
	)

	result := cloneJob(job)

	if err := queue.commitMutationLocked(before); err != nil {
		return Job{}, fmt.Errorf(
			"persist acknowledged job: %w",
			err,
		)
	}

	return result, nil
}

// Nack releases a leased job and persists retry or dead-letter state.
func (queue *Queue) Nack(
	jobID string,
	worker string,
	cause error,
) (Job, error) {
	if queue == nil {
		return Job{}, ErrQueueClosed
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		return Job{}, ErrQueueClosed
	}

	before := queue.snapshotLocked()
	beforeAuditSequence := queue.auditSequence

	now := queue.now()
	queue.recoverExpiredLeasesLocked(now)

	job, err := queue.ownedLeaseLocked(jobID, worker)
	if err != nil {
		if queue.auditSequence != beforeAuditSequence {
			if persistErr := queue.commitMutationLocked(before); persistErr != nil {
				return Job{}, fmt.Errorf(
					"persist lease recovery: %w",
					persistErr,
				)
			}
		}
		return Job{}, err
	}

	if cause == nil {
		job.LastError = "worker rejected job without error"
	} else {
		job.LastError = cause.Error()
	}

	job.LeaseOwner = ""
	job.LeaseUntil = 0
	job.UpdatedAt = now.UnixNano()

	if job.Attempts >= job.MaxAttempts {
		job.Status = JobStatusDeadLetter

		queue.appendAuditLocked(
			job,
			AuditActionDeadLettered,
			worker,
			job.LastError,
		)
	} else {
		job.Status = JobStatusPending
		job.AvailableAt = now.Add(
			queue.retryBackoffLocked(job.Attempts),
		).UnixNano()

		queue.appendAuditLocked(
			job,
			AuditActionRetryScheduled,
			worker,
			job.LastError,
		)
	}

	result := cloneJob(job)

	if err := queue.commitMutationLocked(before); err != nil {
		return Job{}, fmt.Errorf(
			"persist rejected job: %w",
			err,
		)
	}

	if job.Status == JobStatusPending {
		queue.signalLocked()
	}

	return result, nil
}

func (queue *Queue) Job(jobID string) (Job, bool) {
	if queue == nil {
		return Job{}, false
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.recoverExpiredLeasesLocked(queue.now())

	job := queue.jobs[jobID]
	if job == nil {
		return Job{}, false
	}

	return cloneJob(job), true
}

func (queue *Queue) DeadLetters() []Job {
	if queue == nil {
		return []Job{}
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	queue.recoverExpiredLeasesLocked(queue.now())

	result := make([]Job, 0)

	for _, jobID := range queue.order {
		job := queue.jobs[jobID]
		if job != nil &&
			job.Status == JobStatusDeadLetter {
			result = append(result, cloneJob(job))
		}
	}

	return result
}

func (queue *Queue) AuditSnapshot() []AuditEvent {
	if queue == nil {
		return []AuditEvent{}
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	result := make(
		[]AuditEvent,
		len(queue.audit),
	)

	for index, event := range queue.audit {
		result[index] = cloneAuditEvent(event)
	}

	return result
}

func (queue *Queue) Len() int {
	if queue == nil {
		return 0
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	return len(queue.jobs)
}

func (queue *Queue) Closed() bool {
	if queue == nil {
		return true
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	return queue.closed
}

// ShutdownWithError persists the shutdown audit before closing the queue.
func (queue *Queue) ShutdownWithError(actor string) error {
	if queue == nil {
		return nil
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		return nil
	}

	before := queue.snapshotLocked()

	queue.closed = true
	queue.appendAuditLocked(
		nil,
		AuditActionShutdown,
		strings.TrimSpace(actor),
		"queue shut down",
	)

	if err := queue.commitMutationLocked(before); err != nil {
		queue.closed = false
		return fmt.Errorf(
			"persist queue shutdown: %w",
			err,
		)
	}

	queue.signalLocked()
	return nil
}

// Shutdown remains source-compatible. Persistence errors are fail-closed:
// the queue remains open when shutdown state cannot be stored.
func (queue *Queue) Shutdown(actor string) {
	_ = queue.ShutdownWithError(actor)
}

func (queue *Queue) claimableJobLocked(
	queueName string,
	now time.Time,
) *Job {
	nowUnix := now.UnixNano()

	for _, jobID := range queue.order {
		job := queue.jobs[jobID]
		if job == nil ||
			job.Queue != queueName ||
			job.Status != JobStatusPending ||
			job.AvailableAt > nowUnix {
			continue
		}

		if job.RequiresApproval &&
			job.ApprovedBy == "" {
			continue
		}

		return job
	}

	return nil
}

func (queue *Queue) ownedLeaseLocked(
	jobID string,
	worker string,
) (*Job, error) {
	jobID = strings.TrimSpace(jobID)
	worker = strings.TrimSpace(worker)

	if jobID == "" || worker == "" {
		return nil, errors.New(
			"job ID and lease worker required",
		)
	}

	job := queue.jobs[jobID]
	if job == nil {
		return nil, ErrJobNotFound
	}

	if job.Status != JobStatusLeased ||
		job.LeaseOwner != worker {
		return nil, ErrLeaseLost
	}

	return job, nil
}

func (queue *Queue) recoverExpiredLeasesLocked(
	now time.Time,
) {
	nowUnix := now.UnixNano()

	for _, jobID := range queue.order {
		job := queue.jobs[jobID]
		if job == nil ||
			job.Status != JobStatusLeased ||
			job.LeaseUntil > nowUnix {
			continue
		}

		previousOwner := job.LeaseOwner
		job.LeaseOwner = ""
		job.LeaseUntil = 0
		job.UpdatedAt = nowUnix

		queue.appendAuditLocked(
			job,
			AuditActionLeaseExpired,
			previousOwner,
			"worker lease expired",
		)

		if job.Attempts >= job.MaxAttempts {
			job.Status = JobStatusDeadLetter
			job.LastError = "worker lease expired"

			queue.appendAuditLocked(
				job,
				AuditActionDeadLettered,
				previousOwner,
				job.LastError,
			)
			continue
		}

		job.Status = JobStatusPending
		job.LastError = "worker lease expired"
		job.AvailableAt = now.Add(
			queue.retryBackoffLocked(job.Attempts),
		).UnixNano()

		queue.appendAuditLocked(
			job,
			AuditActionRetryScheduled,
			previousOwner,
			job.LastError,
		)
		queue.signalLocked()
	}
}

func (queue *Queue) nextWakeLocked(
	queueName string,
	now time.Time,
) time.Duration {
	nowUnix := now.UnixNano()
	var earliest int64

	for _, jobID := range queue.order {
		job := queue.jobs[jobID]
		if job == nil || job.Queue != queueName {
			continue
		}

		var candidate int64

		switch job.Status {
		case JobStatusPending:
			if job.RequiresApproval &&
				job.ApprovedBy == "" {
				continue
			}
			candidate = job.AvailableAt

		case JobStatusLeased:
			candidate = job.LeaseUntil

		default:
			continue
		}

		if candidate <= nowUnix {
			return 0
		}

		if earliest == 0 || candidate < earliest {
			earliest = candidate
		}
	}

	if earliest == 0 {
		return -1
	}

	return time.Duration(earliest - nowUnix)
}

func (queue *Queue) retryBackoffLocked(
	attempts int,
) time.Duration {
	exponent := attempts - 1
	if exponent < 0 {
		exponent = 0
	}
	if exponent > 20 {
		exponent = 20
	}

	multiplier := time.Duration(1 << exponent)

	if queue.config.BackoffBase >
		maximumRetryBackoffV1/multiplier {
		return maximumRetryBackoffV1
	}

	backoff := queue.config.BackoffBase * multiplier
	if backoff > maximumRetryBackoffV1 {
		return maximumRetryBackoffV1
	}

	return backoff
}

func (queue *Queue) appendAuditLocked(
	job *Job,
	action AuditAction,
	actor string,
	message string,
) {
	queue.auditSequence++

	event := AuditEvent{
		Sequence: queue.auditSequence,
		Action:   action,
		Actor:    actor,
		Message:  message,
		Time:     queue.now().UnixNano(),
	}

	if job != nil {
		event.JobID = job.ID
		event.Queue = job.Queue
	}

	queue.audit = append(queue.audit, event)

	if len(queue.audit) > queue.config.AuditLimit {
		start := len(queue.audit) - queue.config.AuditLimit
		trimmed := make(
			[]AuditEvent,
			queue.config.AuditLimit,
		)
		copy(trimmed, queue.audit[start:])
		queue.audit = trimmed
	}
}

func (queue *Queue) signalLocked() {
	select {
	case queue.notify <- struct{}{}:
	default:
	}
}

func compoundIdempotencyKeyV1(
	queueName string,
	idempotencyKey string,
) string {
	return queueName + "\x00" + idempotencyKey
}
