package queueworker

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueueEnqueueIsIdempotent(t *testing.T) {
	queue := newTestQueue(t)

	request := EnqueueRequest{
		ID:             "job-1",
		Queue:          "monitoring_alert_queue",
		Payload:        json.RawMessage(`{"alert":"disk"}`),
		IdempotencyKey: "alert-disk-1",
	}

	first, created, err := queue.Enqueue(request)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("first enqueue must create job")
	}

	request.ID = "job-2"

	second, created, err := queue.Enqueue(request)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("duplicate idempotency key must not create job")
	}
	if second.ID != first.ID {
		t.Fatalf(
			"duplicate returned job %q; want %q",
			second.ID,
			first.ID,
		)
	}
	if queue.Len() != 1 {
		t.Fatalf("queue length=%d; want=1", queue.Len())
	}
}

func TestQueueConcurrentEnqueueIsIdempotent(t *testing.T) {
	queue := newTestQueue(t)

	const goroutines = 32

	var createdCount atomic.Int32
	var wait sync.WaitGroup
	wait.Add(goroutines)

	for index := 0; index < goroutines; index++ {
		go func(index int) {
			defer wait.Done()

			_, created, err := queue.Enqueue(
				EnqueueRequest{
					ID: "job-concurrent-" +
						time.Unix(
							int64(index),
							0,
						).Format("150405"),
					Queue: "monitoring_alert_queue",
					Payload: json.RawMessage(
						`{"event":"same"}`,
					),
					IdempotencyKey: "same-event",
				},
			)
			if err != nil {
				t.Errorf("enqueue: %v", err)
				return
			}

			if created {
				createdCount.Add(1)
			}
		}(index)
	}

	wait.Wait()

	if createdCount.Load() != 1 {
		t.Fatalf(
			"created jobs=%d; want=1",
			createdCount.Load(),
		)
	}

	if queue.Len() != 1 {
		t.Fatalf("queue length=%d; want=1", queue.Len())
	}
}

func TestQueueAdminApprovalBlocksClaim(t *testing.T) {
	queue := newTestQueue(t)

	job, created, err := queue.Enqueue(
		EnqueueRequest{
			ID:             "review-1",
			Queue:          "tgwt_withdraw_review_queue",
			Payload:        json.RawMessage(`{"claim":"1"}`),
			IdempotencyKey: "withdraw-1",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("job must be created")
	}
	if !job.RequiresApproval {
		t.Fatal("review queue must require approval")
	}

	cancelled, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	if _, err := queue.Claim(
		cancelled,
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-1",
		},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"unapproved claim error=%v; want context canceled",
			err,
		)
	}

	approved, err := queue.Approve(job.ID, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	if approved.ApprovedBy != "admin-1" {
		t.Fatalf(
			"approved by=%q; want admin-1",
			approved.ApprovedBy,
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
	if claimed.Status != JobStatusLeased {
		t.Fatalf(
			"status=%q; want=%q",
			claimed.Status,
			JobStatusLeased,
		)
	}
}

func TestQueueAckRequiresLeaseOwner(t *testing.T) {
	queue := newTestQueue(t)

	job := enqueueMonitoringJob(
		t,
		queue,
		"ack-job",
		"ack-key",
		2,
	)

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

	if _, err := queue.Ack(
		claimed.ID,
		"worker-2",
	); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf(
			"wrong-owner ack error=%v; want ErrLeaseLost",
			err,
		)
	}

	completed, err := queue.Ack(
		claimed.ID,
		"worker-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != JobStatusSucceeded {
		t.Fatalf(
			"status=%q; want=%q",
			completed.Status,
			JobStatusSucceeded,
		)
	}
}

func TestQueueRenewLeaseRequiresOwner(t *testing.T) {
	queue := newTestQueue(t)

	job := enqueueMonitoringJob(
		t,
		queue,
		"renew-job",
		"renew-key",
		2,
	)

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

	if _, err := queue.RenewLease(
		claimed.ID,
		"worker-2",
	); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf(
			"wrong-owner renewal error=%v; want ErrLeaseLost",
			err,
		)
	}

	renewed, err := queue.RenewLease(
		claimed.ID,
		"worker-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if renewed.LeaseUntil < claimed.LeaseUntil {
		t.Fatal("renewed lease must not move backwards")
	}
}

func TestQueueRetryBackoffAndDeadLetter(t *testing.T) {
	config := DefaultRuntimeConfig()
	config.LeaseDuration = time.Second
	config.BackoffBase = 2 * time.Second
	config.MaxAttempts = 2

	queue, err := NewQueue(config)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_000, 0)
	queue.now = func() time.Time {
		return now
	}

	job := enqueueMonitoringJob(
		t,
		queue,
		"retry-job",
		"retry-key",
		2,
	)

	first, err := queue.Claim(
		context.Background(),
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-1",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	retry, err := queue.Nack(
		first.ID,
		"worker-1",
		errors.New("temporary failure"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if retry.Status != JobStatusPending {
		t.Fatalf(
			"retry status=%q; want pending",
			retry.Status,
		)
	}
	if retry.AvailableAt <= now.UnixNano() {
		t.Fatal("retry must be delayed by backoff")
	}

	cancelled, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	if _, err := queue.Claim(
		cancelled,
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-1",
		},
	); !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"early retry claim error=%v; want context canceled",
			err,
		)
	}

	now = now.Add(2 * time.Second)

	second, err := queue.Claim(
		context.Background(),
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-1",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if second.Attempts != 2 {
		t.Fatalf(
			"attempts=%d; want=2",
			second.Attempts,
		)
	}

	dead, err := queue.Nack(
		second.ID,
		"worker-1",
		errors.New("permanent failure"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if dead.Status != JobStatusDeadLetter {
		t.Fatalf(
			"status=%q; want dead-letter",
			dead.Status,
		)
	}

	deadLetters := queue.DeadLetters()
	if len(deadLetters) != 1 ||
		deadLetters[0].ID != job.ID {
		t.Fatalf(
			"unexpected dead letters: %+v",
			deadLetters,
		)
	}
}

func TestQueueExpiredLeaseIsRecovered(t *testing.T) {
	config := DefaultRuntimeConfig()
	config.LeaseDuration = time.Second
	config.BackoffBase = time.Second
	config.MaxAttempts = 3

	queue, err := NewQueue(config)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(2_000, 0)
	queue.now = func() time.Time {
		return now
	}

	job := enqueueMonitoringJob(
		t,
		queue,
		"lease-job",
		"lease-key",
		3,
	)

	first, err := queue.Claim(
		context.Background(),
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-dead",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(config.LeaseDuration + time.Nanosecond)

	recovered, exists := queue.Job(first.ID)
	if !exists {
		t.Fatal("expired job disappeared")
	}
	if recovered.Status != JobStatusPending {
		t.Fatalf(
			"recovered status=%q; want pending",
			recovered.Status,
		)
	}
	if recovered.LeaseOwner != "" {
		t.Fatal("expired lease owner must be cleared")
	}

	now = now.Add(config.BackoffBase)

	second, err := queue.Claim(
		context.Background(),
		ClaimRequest{
			Queue:  job.Queue,
			Worker: "worker-live",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if second.Attempts != 2 {
		t.Fatalf(
			"recovered attempts=%d; want=2",
			second.Attempts,
		)
	}
}

func TestQueueShutdownUnblocksClaim(t *testing.T) {
	queue := newTestQueue(t)

	result := make(chan error, 1)

	go func() {
		_, err := queue.Claim(
			context.Background(),
			ClaimRequest{
				Queue:  "monitoring_alert_queue",
				Worker: "worker-1",
			},
		)
		result <- err
	}()

	time.Sleep(20 * time.Millisecond)
	queue.Shutdown("test")

	select {
	case err := <-result:
		if !errors.Is(err, ErrQueueClosed) {
			t.Fatalf(
				"claim error=%v; want ErrQueueClosed",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("shutdown did not unblock claim")
	}

	if !queue.Closed() {
		t.Fatal("queue must report closed")
	}

	if _, _, err := queue.Enqueue(
		EnqueueRequest{
			ID:             "after-shutdown",
			Queue:          "monitoring_alert_queue",
			Payload:        json.RawMessage(`{"x":1}`),
			IdempotencyKey: "after-shutdown",
		},
	); !errors.Is(err, ErrQueueClosed) {
		t.Fatalf(
			"enqueue error=%v; want ErrQueueClosed",
			err,
		)
	}
}

func TestQueueAuditIsBounded(t *testing.T) {
	config := DefaultRuntimeConfig()
	config.AuditLimit = 3

	queue, err := NewQueue(config)
	if err != nil {
		t.Fatal(err)
	}

	request := EnqueueRequest{
		ID:             "audit-job",
		Queue:          "monitoring_alert_queue",
		Payload:        json.RawMessage(`{"audit":true}`),
		IdempotencyKey: "audit-key",
	}

	if _, _, err := queue.Enqueue(request); err != nil {
		t.Fatal(err)
	}

	for index := 0; index < 10; index++ {
		request.ID = "ignored-duplicate"
		if _, _, err := queue.Enqueue(request); err != nil {
			t.Fatal(err)
		}
	}

	audit := queue.AuditSnapshot()
	if len(audit) != config.AuditLimit {
		t.Fatalf(
			"audit length=%d; want=%d",
			len(audit),
			config.AuditLimit,
		)
	}

	for index := 1; index < len(audit); index++ {
		if audit[index].Sequence <=
			audit[index-1].Sequence {
			t.Fatal(
				"audit sequence must increase",
			)
		}
	}
}

func TestQueueSnapshotsDetachPayload(t *testing.T) {
	queue := newTestQueue(t)

	job := enqueueMonitoringJob(
		t,
		queue,
		"detach-job",
		"detach-key",
		2,
	)

	job.Payload[0] = 'X'

	stored, exists := queue.Job(job.ID)
	if !exists {
		t.Fatal("stored job missing")
	}
	if !json.Valid(stored.Payload) {
		t.Fatal("caller mutation changed stored payload")
	}
}

func newTestQueue(t *testing.T) *Queue {
	t.Helper()

	config := DefaultRuntimeConfig()
	config.LeaseDuration = 100 * time.Millisecond
	config.BackoffBase = time.Millisecond
	config.MaxAttempts = 3
	config.AuditLimit = 100

	queue, err := NewQueue(config)
	if err != nil {
		t.Fatal(err)
	}

	return queue
}

func enqueueMonitoringJob(
	t *testing.T,
	queue *Queue,
	jobID string,
	idempotencyKey string,
	maxAttempts int,
) Job {
	t.Helper()

	job, created, err := queue.Enqueue(
		EnqueueRequest{
			ID:             jobID,
			Queue:          "monitoring_alert_queue",
			Payload:        json.RawMessage(`{"value":1}`),
			IdempotencyKey: idempotencyKey,
			MaxAttempts:    maxAttempts,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected new job")
	}

	return job
}
