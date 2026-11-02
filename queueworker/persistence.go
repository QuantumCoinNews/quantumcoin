package queueworker

import (
	"errors"
	"fmt"
)

// NewPersistentQueue constructs a queue from validated durable state.
// A missing snapshot creates an empty queue. A corrupt or policy-incompatible
// snapshot fails closed.
func NewPersistentQueue(
	config RuntimeConfig,
	store StateStore,
) (*Queue, error) {
	if store == nil {
		return nil, errors.New(
			"persistent queue state store required",
		)
	}

	queue, err := NewQueue(config)
	if err != nil {
		return nil, err
	}

	snapshot, exists, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf(
			"load persistent queue state: %w",
			err,
		)
	}

	queue.store = store

	if !exists {
		return queue, nil
	}

	if err := validateSnapshotAgainstPlanV1(
		snapshot,
		config.Plan,
		config.AuditLimit,
	); err != nil {
		return nil, fmt.Errorf(
			"validate persistent queue policy: %w",
			err,
		)
	}

	queue.restoreSnapshotLocked(snapshot)

	return queue, nil
}

func validateSnapshotAgainstPlanV1(
	snapshot QueueSnapshotV1,
	plan Plan,
	auditLimit int,
) error {
	if err := ValidateQueueSnapshotV1(snapshot); err != nil {
		return err
	}

	if len(snapshot.Audit) > auditLimit {
		return fmt.Errorf(
			"snapshot audit length exceeds runtime limit: "+
				"got=%d limit=%d",
			len(snapshot.Audit),
			auditLimit,
		)
	}

	for jobID, job := range snapshot.Jobs {
		policy, exists := queuePolicyByName(
			plan,
			job.Queue,
		)
		if !exists {
			return fmt.Errorf(
				"snapshot job %s uses unknown queue: %s",
				jobID,
				job.Queue,
			)
		}

		if job.RequiresApproval !=
			policy.RequiresAdminApproval {
			return fmt.Errorf(
				"snapshot approval policy mismatch for job: %s",
				jobID,
			)
		}

		if job.Status == JobStatusSucceeded &&
			job.LastError != "" {
			return fmt.Errorf(
				"succeeded snapshot job retains error: %s",
				jobID,
			)
		}
	}

	return nil
}

func (queue *Queue) snapshotLocked() QueueSnapshotV1 {
	snapshot := QueueSnapshotV1{
		Version: QueueSnapshotVersionV1,
		Jobs: make(
			map[string]Job,
			len(queue.jobs),
		),
		Order: append(
			[]string(nil),
			queue.order...,
		),
		Idempotency: make(
			map[string]string,
			len(queue.idempotency),
		),
		Audit: append(
			[]AuditEvent(nil),
			queue.audit...,
		),
		AuditSequence: queue.auditSequence,
	}

	for jobID, job := range queue.jobs {
		if job == nil {
			continue
		}

		snapshot.Jobs[jobID] = cloneJob(job)
	}

	for key, jobID := range queue.idempotency {
		snapshot.Idempotency[key] = jobID
	}

	return snapshot
}

func (queue *Queue) restoreSnapshotLocked(
	snapshot QueueSnapshotV1,
) {
	queue.jobs = make(
		map[string]*Job,
		len(snapshot.Jobs),
	)

	for jobID, storedJob := range snapshot.Jobs {
		job := storedJob
		job.Payload = append(
			[]byte(nil),
			storedJob.Payload...,
		)
		queue.jobs[jobID] = &job
	}

	queue.order = append(
		[]string(nil),
		snapshot.Order...,
	)

	queue.idempotency = make(
		map[string]string,
		len(snapshot.Idempotency),
	)
	for key, jobID := range snapshot.Idempotency {
		queue.idempotency[key] = jobID
	}

	queue.audit = append(
		[]AuditEvent(nil),
		snapshot.Audit...,
	)
	queue.auditSequence = snapshot.AuditSequence
}

func (queue *Queue) persistLocked() error {
	if queue.store == nil {
		return nil
	}

	snapshot := queue.snapshotLocked()

	if err := queue.store.Save(snapshot); err != nil {
		return fmt.Errorf(
			"persist queue state: %w",
			err,
		)
	}

	return nil
}

// commitMutationLocked saves one completed in-memory mutation. If durable
// storage rejects the new state, the prior snapshot is restored before the
// error is returned.
func (queue *Queue) commitMutationLocked(
	before QueueSnapshotV1,
) error {
	if err := queue.persistLocked(); err != nil {
		queue.restoreSnapshotLocked(before)

		return err
	}

	return nil
}

// Checkpoint persists the current state without mutating it.
func (queue *Queue) Checkpoint() error {
	if queue == nil {
		return ErrQueueClosed
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.closed {
		return ErrQueueClosed
	}

	return queue.persistLocked()
}
