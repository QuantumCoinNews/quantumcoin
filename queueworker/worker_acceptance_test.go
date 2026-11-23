package queueworker

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestStandaloneWorkerExplicitStartDurableReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testnet", "queue.json")
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultRuntimeConfig()
	queue, err := NewPersistentQueue(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	job := enqueueMonitoringJob(t, queue, "receipt-job", "receipt-key", 2)
	var calls atomic.Int32
	worker, err := NewWorker(queue, "lab-worker", job.Queue, func(context.Context, Job) error { calls.Add(1); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if worker.Running() || calls.Load() != 0 {
		t.Fatal("construction executed a handler")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer func() {
		if worker.Running() {
			_ = worker.Stop(ctx)
		}
	}()
	if err := worker.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(worker.Start(ctx), ErrWorkerAlreadyStarted) {
		t.Fatal("duplicate start accepted")
	}
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		current, ok := queue.Job(job.ID)
		if ok && current.Status == JobStatusSucceeded {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("no durable receipt")
		case <-ticker.C:
		}
	}
	if err := worker.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if worker.Running() || calls.Load() != 1 {
		t.Fatal("worker lifecycle or delivery count mismatch")
	}
	restored, err := NewPersistentQueue(cfg, store)
	if err != nil {
		t.Fatal(err)
	}
	current, ok := restored.Job(job.ID)
	if !ok || current.Status != JobStatusSucceeded {
		t.Fatal("ack was not persisted")
	}
	if _, created, err := restored.Enqueue(EnqueueRequest{ID: "repeat", Queue: job.Queue, Payload: job.Payload, IdempotencyKey: job.IdempotencyKey}); err != nil || created {
		t.Fatalf("restart idempotency lost: %v", err)
	}
}

func TestStandaloneWorkerPlanOnlyCannotStart(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	cfg.Plan = DefaultPlanV1()
	queue, err := NewQueue(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewWorker(queue, "lab", "monitoring_alert_queue", func(context.Context, Job) error { return nil }); err == nil {
		t.Fatal("plan-only worker allowed")
	}
}
