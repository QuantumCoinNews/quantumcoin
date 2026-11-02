package queueworker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPersistentQueueCheckpointRestoresState(
	t *testing.T,
) {
	storePath := filepath.Join(
		t.TempDir(),
		"queue",
		"state.json",
	)

	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultRuntimeConfig()

	first, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	enqueued, created, err := first.Enqueue(
		EnqueueRequest{
			ID:             "persistent-job-1",
			Queue:          "monitoring_alert_queue",
			Payload:        json.RawMessage(`{"event":"restart"}`),
			IdempotencyKey: "restart-event-1",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("first enqueue must create job")
	}

	if err := first.Checkpoint(); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	second, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatalf("restore queue: %v", err)
	}

	restored, exists := second.Job(enqueued.ID)
	if !exists {
		t.Fatal("checkpointed job was not restored")
	}
	if restored.ID != enqueued.ID {
		t.Fatalf(
			"restored job=%q; want=%q",
			restored.ID,
			enqueued.ID,
		)
	}

	duplicateRequest := EnqueueRequest{
		ID:             "persistent-job-2",
		Queue:          enqueued.Queue,
		Payload:        json.RawMessage(`{"event":"restart"}`),
		IdempotencyKey: enqueued.IdempotencyKey,
	}

	duplicate, created, err := second.Enqueue(
		duplicateRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal(
			"restored idempotency key created duplicate job",
		)
	}
	if duplicate.ID != enqueued.ID {
		t.Fatalf(
			"duplicate returned=%q; want=%q",
			duplicate.ID,
			enqueued.ID,
		)
	}
}

func TestPersistentQueueStartsEmptyWithoutSnapshot(
	t *testing.T,
) {
	store, err := NewFileStore(
		filepath.Join(
			t.TempDir(),
			"queue",
			"state.json",
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	queue, err := NewPersistentQueue(
		DefaultRuntimeConfig(),
		store,
	)
	if err != nil {
		t.Fatal(err)
	}

	if queue.Len() != 0 {
		t.Fatalf(
			"new persistent queue length=%d; want=0",
			queue.Len(),
		)
	}
}

func TestPersistentQueueRejectsCorruptSnapshot(
	t *testing.T,
) {
	storePath := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	if err := os.WriteFile(
		storePath,
		[]byte("{corrupt"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewPersistentQueue(
		DefaultRuntimeConfig(),
		store,
	); err == nil {
		t.Fatal("corrupt persistent state must fail")
	}
}

func TestPersistentQueueRejectsUnknownQueueInSnapshot(
	t *testing.T,
) {
	store, err := NewFileStore(
		filepath.Join(
			t.TempDir(),
			"state.json",
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := validQueueSnapshotV1()
	job := snapshot.Jobs["job-1"]
	job.Queue = "unknown_queue"
	snapshot.Jobs["job-1"] = job

	snapshot.Idempotency = map[string]string{
		compoundIdempotencyKeyV1(
			job.Queue,
			job.IdempotencyKey,
		): job.ID,
	}

	if err := store.Save(snapshot); err != nil {
		t.Fatalf(
			"store structurally valid snapshot: %v",
			err,
		)
	}

	if _, err := NewPersistentQueue(
		DefaultRuntimeConfig(),
		store,
	); err == nil {
		t.Fatal(
			"snapshot with unknown policy queue must fail",
		)
	}
}

func TestPersistentQueueRejectsApprovalPolicyMismatch(
	t *testing.T,
) {
	store, err := NewFileStore(
		filepath.Join(
			t.TempDir(),
			"state.json",
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := validQueueSnapshotV1()
	job := snapshot.Jobs["job-1"]
	job.RequiresApproval = true
	snapshot.Jobs["job-1"] = job

	if err := store.Save(snapshot); err != nil {
		t.Fatal(err)
	}

	if _, err := NewPersistentQueue(
		DefaultRuntimeConfig(),
		store,
	); err == nil {
		t.Fatal(
			"snapshot approval policy mismatch must fail",
		)
	}
}

func TestPersistentQueueRejectsAuditAboveRuntimeLimit(
	t *testing.T,
) {
	store, err := NewFileStore(
		filepath.Join(
			t.TempDir(),
			"state.json",
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := validQueueSnapshotV1()

	for sequence := uint64(2); sequence <= 5; sequence++ {
		snapshot.Audit = append(
			snapshot.Audit,
			AuditEvent{
				Sequence: sequence,
				Action:   AuditActionDuplicate,
				Time:     int64(sequence),
			},
		)
		snapshot.AuditSequence = sequence
	}

	if err := store.Save(snapshot); err != nil {
		t.Fatal(err)
	}

	config := DefaultRuntimeConfig()
	config.AuditLimit = 2

	if _, err := NewPersistentQueue(
		config,
		store,
	); err == nil {
		t.Fatal(
			"snapshot above runtime audit limit must fail",
		)
	}
}

func TestPersistentQueueCheckpointPropagatesStoreFailure(
	t *testing.T,
) {
	store := &failingStateStore{
		saveErr: errors.New("disk unavailable"),
	}

	queue, err := NewPersistentQueue(
		DefaultRuntimeConfig(),
		store,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := queue.Checkpoint(); err == nil {
		t.Fatal("checkpoint store failure must propagate")
	}
}

type failingStateStore struct {
	snapshot QueueSnapshotV1
	exists   bool
	loadErr  error
	saveErr  error
}

func (store *failingStateStore) Load() (
	QueueSnapshotV1,
	bool,
	error,
) {
	if store.loadErr != nil {
		return QueueSnapshotV1{}, false, store.loadErr
	}

	return cloneQueueSnapshotV1(store.snapshot),
		store.exists,
		nil
}

func (store *failingStateStore) Save(
	snapshot QueueSnapshotV1,
) error {
	if store.saveErr != nil {
		return store.saveErr
	}

	store.snapshot = cloneQueueSnapshotV1(snapshot)
	store.exists = true
	return nil
}
func TestPersistentQueueAutomaticallyRestoresEnqueue(
	t *testing.T,
) {
	store, err := NewFileStore(
		filepath.Join(
			t.TempDir(),
			"state.json",
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultRuntimeConfig()

	first, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	job, created, err := first.Enqueue(
		EnqueueRequest{
			ID:             "automatic-enqueue",
			Queue:          "monitoring_alert_queue",
			Payload:        json.RawMessage(`{"automatic":true}`),
			IdempotencyKey: "automatic-enqueue-key",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("enqueue must create job")
	}

	second, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	restored, exists := second.Job(job.ID)
	if !exists {
		t.Fatal(
			"enqueue was not restored without checkpoint",
		)
	}
	if restored.IdempotencyKey != job.IdempotencyKey {
		t.Fatal(
			"restored enqueue lost idempotency key",
		)
	}
}

func TestPersistentQueueAutomaticallyRestoresApproval(
	t *testing.T,
) {
	store, err := NewFileStore(
		filepath.Join(
			t.TempDir(),
			"state.json",
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultRuntimeConfig()

	first, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	job, created, err := first.Enqueue(
		EnqueueRequest{
			ID:             "automatic-approval",
			Queue:          "tgwt_withdraw_review_queue",
			Payload:        json.RawMessage(`{"review":true}`),
			IdempotencyKey: "automatic-approval-key",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("approval job must be created")
	}

	if _, err := first.Approve(
		job.ID,
		"admin-1",
	); err != nil {
		t.Fatal(err)
	}

	second, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	restored, exists := second.Job(job.ID)
	if !exists {
		t.Fatal("approved job was not restored")
	}
	if restored.ApprovedBy != "admin-1" {
		t.Fatalf(
			"approved by=%q; want admin-1",
			restored.ApprovedBy,
		)
	}
	if restored.ApprovedAt <= 0 {
		t.Fatal("approval timestamp was not restored")
	}
}

func TestPersistentQueueEnqueueRollsBackOnStoreFailure(
	t *testing.T,
) {
	store := &failingStateStore{
		saveErr: errors.New("disk unavailable"),
	}

	queue, err := NewPersistentQueue(
		DefaultRuntimeConfig(),
		store,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, created, err := queue.Enqueue(
		EnqueueRequest{
			ID:             "rollback-enqueue",
			Queue:          "monitoring_alert_queue",
			Payload:        json.RawMessage(`{"rollback":true}`),
			IdempotencyKey: "rollback-enqueue-key",
		},
	)
	if err == nil {
		t.Fatal("store failure must fail enqueue")
	}
	if created {
		t.Fatal("failed durable enqueue cannot report created")
	}
	if queue.Len() != 0 {
		t.Fatalf(
			"queue length=%d; want=0 after rollback",
			queue.Len(),
		)
	}
}

func TestPersistentQueueApprovalRollsBackOnStoreFailure(
	t *testing.T,
) {
	store := &failingStateStore{}

	queue, err := NewPersistentQueue(
		DefaultRuntimeConfig(),
		store,
	)
	if err != nil {
		t.Fatal(err)
	}

	job, created, err := queue.Enqueue(
		EnqueueRequest{
			ID:             "rollback-approval",
			Queue:          "tgwt_withdraw_review_queue",
			Payload:        json.RawMessage(`{"approval":true}`),
			IdempotencyKey: "rollback-approval-key",
		},
	)
	if err != nil || !created {
		t.Fatalf(
			"enqueue created=%v err=%v",
			created,
			err,
		)
	}

	store.saveErr = errors.New("disk unavailable")

	if _, err := queue.Approve(
		job.ID,
		"admin-1",
	); err == nil {
		t.Fatal("store failure must fail approval")
	}

	restored, exists := queue.Job(job.ID)
	if !exists {
		t.Fatal("rolled-back job missing")
	}
	if restored.ApprovedBy != "" ||
		restored.ApprovedAt != 0 {
		t.Fatal(
			"failed approval remained in memory",
		)
	}
}
func TestPersistentQueueAutomaticallyRestoresClaim(
	t *testing.T,
) {
	store, err := NewFileStore(
		filepath.Join(t.TempDir(), "state.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultRuntimeConfig()
	config.LeaseDuration = time.Minute

	first, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	job, created, err := first.Enqueue(
		EnqueueRequest{
			ID:             "persistent-claim",
			Queue:          "monitoring_alert_queue",
			Payload:        json.RawMessage(`{"claim":true}`),
			IdempotencyKey: "persistent-claim-key",
		},
	)
	if err != nil || !created {
		t.Fatalf(
			"enqueue created=%v err=%v",
			created,
			err,
		)
	}

	claimed, err := first.Claim(
		context.Background(),
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-1",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	second, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	restored, exists := second.Job(job.ID)
	if !exists {
		t.Fatal("claimed job was not restored")
	}
	if restored.Status != JobStatusLeased {
		t.Fatalf(
			"status=%q; want leased",
			restored.Status,
		)
	}
	if restored.LeaseOwner != "worker-1" {
		t.Fatalf(
			"owner=%q; want worker-1",
			restored.LeaseOwner,
		)
	}
	if restored.LeaseUntil != claimed.LeaseUntil {
		t.Fatalf(
			"lease=%d; want=%d",
			restored.LeaseUntil,
			claimed.LeaseUntil,
		)
	}
	if restored.Attempts != 1 {
		t.Fatalf(
			"attempts=%d; want=1",
			restored.Attempts,
		)
	}
}

func TestPersistentQueueClaimRollsBackOnStoreFailure(
	t *testing.T,
) {
	store := &failingStateStore{}

	config := DefaultRuntimeConfig()
	config.LeaseDuration = time.Minute

	queue, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	job, created, err := queue.Enqueue(
		EnqueueRequest{
			ID:             "claim-rollback",
			Queue:          "monitoring_alert_queue",
			Payload:        json.RawMessage(`{"rollback":true}`),
			IdempotencyKey: "claim-rollback-key",
		},
	)
	if err != nil || !created {
		t.Fatalf(
			"enqueue created=%v err=%v",
			created,
			err,
		)
	}

	store.saveErr = errors.New("disk unavailable")

	_, err = queue.Claim(
		context.Background(),
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-1",
		},
	)
	if err == nil {
		t.Fatal("store failure must fail claim")
	}

	restored, exists := queue.Job(job.ID)
	if !exists {
		t.Fatal("rolled-back claim job missing")
	}
	if restored.Status != JobStatusPending {
		t.Fatalf(
			"status=%q; want pending",
			restored.Status,
		)
	}
	if restored.LeaseOwner != "" ||
		restored.LeaseUntil != 0 {
		t.Fatal("failed claim retained lease")
	}
	if restored.Attempts != 0 {
		t.Fatalf(
			"attempts=%d; want=0",
			restored.Attempts,
		)
	}
}

func TestPersistentQueueAutomaticallyRestoresLeaseRenewal(
	t *testing.T,
) {
	store, err := NewFileStore(
		filepath.Join(t.TempDir(), "state.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultRuntimeConfig()
	config.LeaseDuration = time.Minute

	first, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	job, created, err := first.Enqueue(
		EnqueueRequest{
			ID:             "persistent-renewal",
			Queue:          "monitoring_alert_queue",
			Payload:        json.RawMessage(`{"renew":true}`),
			IdempotencyKey: "persistent-renewal-key",
		},
	)
	if err != nil || !created {
		t.Fatalf(
			"enqueue created=%v err=%v",
			created,
			err,
		)
	}

	claimed, err := first.Claim(
		context.Background(),
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-1",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond)

	renewed, err := first.RenewLease(
		job.ID,
		"worker-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.LeaseUntil <= claimed.LeaseUntil {
		t.Fatal("renewed lease did not move forward")
	}

	second, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	restored, exists := second.Job(job.ID)
	if !exists {
		t.Fatal("renewed job was not restored")
	}
	if restored.LeaseUntil != renewed.LeaseUntil {
		t.Fatalf(
			"restored lease=%d; want=%d",
			restored.LeaseUntil,
			renewed.LeaseUntil,
		)
	}

	foundAudit := false
	for _, event := range second.AuditSnapshot() {
		if event.JobID == job.ID &&
			event.Action == AuditActionLeaseRenewed {
			foundAudit = true
			break
		}
	}

	if !foundAudit {
		t.Fatal("lease renewal audit was not restored")
	}
}

func TestPersistentQueueLeaseRenewalRollsBackOnStoreFailure(
	t *testing.T,
) {
	store := &failingStateStore{}

	config := DefaultRuntimeConfig()
	config.LeaseDuration = time.Minute

	queue, err := NewPersistentQueue(config, store)
	if err != nil {
		t.Fatal(err)
	}

	job, created, err := queue.Enqueue(
		EnqueueRequest{
			ID:             "renewal-rollback",
			Queue:          "monitoring_alert_queue",
			Payload:        json.RawMessage(`{"renew":false}`),
			IdempotencyKey: "renewal-rollback-key",
		},
	)
	if err != nil || !created {
		t.Fatalf(
			"enqueue created=%v err=%v",
			created,
			err,
		)
	}

	claimed, err := queue.Claim(
		context.Background(),
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-1",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	store.saveErr = errors.New("disk unavailable")
	time.Sleep(time.Millisecond)

	if _, err := queue.RenewLease(
		job.ID,
		"worker-1",
	); err == nil {
		t.Fatal("store failure must fail lease renewal")
	}

	restored, exists := queue.Job(job.ID)
	if !exists {
		t.Fatal("rolled-back leased job missing")
	}
	if restored.LeaseUntil != claimed.LeaseUntil {
		t.Fatalf(
			"lease changed after rollback: got=%d want=%d",
			restored.LeaseUntil,
			claimed.LeaseUntil,
		)
	}
}
