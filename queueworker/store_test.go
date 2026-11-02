package queueworker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"queue",
		"state.json",
	)

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := validQueueSnapshotV1()

	if err := store.Save(snapshot); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	loaded, exists, err := store.Load()
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if !exists {
		t.Fatal("saved snapshot must exist")
	}

	if loaded.Version != QueueSnapshotVersionV1 {
		t.Fatalf(
			"version=%q; want=%q",
			loaded.Version,
			QueueSnapshotVersionV1,
		)
	}

	job := loaded.Jobs["job-1"]
	if job.ID != "job-1" {
		t.Fatalf("loaded job=%q; want job-1", job.ID)
	}

	if loaded.Idempotency[compoundIdempotencyKeyV1(
		job.Queue,
		job.IdempotencyKey,
	)] != job.ID {
		t.Fatal("idempotency index was not restored")
	}

	if len(loaded.Audit) != 1 {
		t.Fatalf(
			"audit length=%d; want=1",
			len(loaded.Audit),
		)
	}
}

func TestFileStoreMissingSnapshot(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"missing",
		"state.json",
	)

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}

	snapshot, exists, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("missing snapshot must not exist")
	}
	if snapshot.Version != "" {
		t.Fatal("missing snapshot must return zero value")
	}
}

func TestFileStoreRejectsRelativePath(t *testing.T) {
	if _, err := NewFileStore(
		filepath.Join("relative", "state.json"),
	); err == nil {
		t.Fatal("relative store path must fail")
	}
}

func TestFileStoreRejectsCorruptSnapshot(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	if err := os.WriteFile(
		path,
		[]byte("{not-json"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.Load(); err == nil {
		t.Fatal("corrupt snapshot must fail")
	}
}

func TestFileStoreRejectsUnknownVersion(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	snapshot := validQueueSnapshotV1()
	snapshot.Version = "UNKNOWN_QUEUE_SNAPSHOT"

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		path,
		data,
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.Load(); err == nil {
		t.Fatal("unknown snapshot version must fail")
	}
}

func TestFileStoreRejectsInvalidSnapshotBeforeSave(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := validQueueSnapshotV1()
	snapshot.Jobs["job-1"] = Job{
		ID:     "job-1",
		Status: JobStatus("unknown"),
	}

	if err := store.Save(snapshot); err == nil {
		t.Fatal("invalid snapshot must not be saved")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf(
			"invalid save created target: %v",
			err,
		)
	}
}

func TestFileStoreReplacesExistingSnapshot(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}

	first := validQueueSnapshotV1()
	if err := store.Save(first); err != nil {
		t.Fatal(err)
	}

	second := validQueueSnapshotV1()
	job := second.Jobs["job-1"]
	job.Status = JobStatusSucceeded
	job.UpdatedAt++
	second.Jobs["job-1"] = job

	if err := store.Save(second); err != nil {
		t.Fatal(err)
	}

	loaded, exists, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("replacement snapshot missing")
	}

	if loaded.Jobs["job-1"].Status !=
		JobStatusSucceeded {
		t.Fatalf(
			"status=%q; want succeeded",
			loaded.Jobs["job-1"].Status,
		)
	}

	if _, err := os.Stat(
		path + ".previous",
	); !os.IsNotExist(err) {
		t.Fatalf(
			"previous snapshot was not cleaned: %v",
			err,
		)
	}
}

func TestFileStoreLoadReturnsDetachedSnapshot(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Save(
		validQueueSnapshotV1(),
	); err != nil {
		t.Fatal(err)
	}

	first, exists, err := store.Load()
	if err != nil || !exists {
		t.Fatalf(
			"first load exists=%v err=%v",
			exists,
			err,
		)
	}

	job := first.Jobs["job-1"]
	job.Payload[0] = 'X'
	first.Jobs["job-1"] = job
	first.Order[0] = "changed"
	first.Idempotency["changed"] = "changed"
	first.Audit[0].Message = "changed"

	second, exists, err := store.Load()
	if err != nil || !exists {
		t.Fatalf(
			"second load exists=%v err=%v",
			exists,
			err,
		)
	}

	if !json.Valid(second.Jobs["job-1"].Payload) {
		t.Fatal("loaded payload was not detached")
	}
	if second.Order[0] != "job-1" {
		t.Fatal("loaded order was not detached")
	}
	if _, exists := second.Idempotency["changed"]; exists {
		t.Fatal("loaded idempotency map was not detached")
	}
	if second.Audit[0].Message == "changed" {
		t.Fatal("loaded audit was not detached")
	}
}

func TestFileStoreConcurrentSaveRemainsValid(
	t *testing.T,
) {
	path := filepath.Join(
		t.TempDir(),
		"state.json",
	)

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}

	const writers = 16

	var wait sync.WaitGroup
	wait.Add(writers)

	for index := 0; index < writers; index++ {
		go func(index int) {
			defer wait.Done()

			snapshot := validQueueSnapshotV1()
			job := snapshot.Jobs["job-1"]
			job.UpdatedAt += int64(index)
			snapshot.Jobs["job-1"] = job

			if err := store.Save(snapshot); err != nil {
				t.Errorf(
					"concurrent save %d: %v",
					index,
					err,
				)
			}
		}(index)
	}

	wait.Wait()

	loaded, exists, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("concurrent snapshot missing")
	}

	if err := ValidateQueueSnapshotV1(loaded); err != nil {
		t.Fatalf(
			"concurrent snapshot invalid: %v",
			err,
		)
	}
}

func TestValidateQueueSnapshotRejectsBrokenIndexes(
	t *testing.T,
) {
	tests := []struct {
		name   string
		mutate func(*QueueSnapshotV1)
	}{
		{
			name: "missing-order",
			mutate: func(snapshot *QueueSnapshotV1) {
				snapshot.Order = nil
			},
		},
		{
			name: "duplicate-order",
			mutate: func(snapshot *QueueSnapshotV1) {
				snapshot.Order = []string{
					"job-1",
					"job-1",
				}
			},
		},
		{
			name: "missing-idempotency",
			mutate: func(snapshot *QueueSnapshotV1) {
				snapshot.Idempotency = map[string]string{}
			},
		},
		{
			name: "wrong-idempotency-target",
			mutate: func(snapshot *QueueSnapshotV1) {
				for key := range snapshot.Idempotency {
					snapshot.Idempotency[key] = "missing-job"
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validQueueSnapshotV1()
			test.mutate(&snapshot)

			if err := ValidateQueueSnapshotV1(
				snapshot,
			); err == nil {
				t.Fatal(
					"broken snapshot index must fail",
				)
			}
		})
	}
}

func validQueueSnapshotV1() QueueSnapshotV1 {
	now := time.Now().UnixNano()

	job := Job{
		ID:             "job-1",
		Queue:          "monitoring_alert_queue",
		Payload:        json.RawMessage(`{"event":"test"}`),
		IdempotencyKey: "event-test-1",
		Status:         JobStatusPending,
		MaxAttempts:    3,
		AvailableAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	idempotencyKey := compoundIdempotencyKeyV1(
		job.Queue,
		job.IdempotencyKey,
	)

	return QueueSnapshotV1{
		Version: QueueSnapshotVersionV1,
		Jobs: map[string]Job{
			job.ID: job,
		},
		Order: []string{
			job.ID,
		},
		Idempotency: map[string]string{
			idempotencyKey: job.ID,
		},
		Audit: []AuditEvent{
			{
				Sequence: 1,
				JobID:    job.ID,
				Queue:    job.Queue,
				Action:   AuditActionEnqueued,
				Message:  "test enqueue",
				Time:     now,
			},
		},
		AuditSequence: 1,
	}
}
