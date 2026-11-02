package queueworker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrWorkerAlreadyStarted = errors.New(
		"queue worker already started",
	)
	ErrWorkerNotStarted = errors.New(
		"queue worker not started",
	)
)

type Worker struct {
	queue     *Queue
	id        string
	queueName string
	handler   Handler

	mu      sync.Mutex
	started bool
	running bool
	cancel  context.CancelFunc
	done    chan struct{}
}

func NewWorker(
	queue *Queue,
	id string,
	queueName string,
	handler Handler,
) (*Worker, error) {
	if queue == nil {
		return nil, errors.New("worker queue required")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("worker identity required")
	}

	queueName = strings.TrimSpace(queueName)
	if queueName == "" {
		return nil, errors.New("worker queue name required")
	}

	if handler == nil {
		return nil, errors.New("worker handler required")
	}

	if queue.config.Plan.Mode != QueueModeRuntimeManual {
		return nil, errors.New(
			"worker requires explicit runtime-manual plan",
		)
	}

	if _, exists := queuePolicyByName(
		queue.config.Plan,
		queueName,
	); !exists {
		return nil, fmt.Errorf(
			"unknown worker queue: %s",
			queueName,
		)
	}

	if queue.Closed() {
		return nil, ErrQueueClosed
	}

	return &Worker{
		queue:     queue,
		id:        id,
		queueName: queueName,
		handler:   handler,
		done:      make(chan struct{}),
	}, nil
}

// Start explicitly starts the worker. Construction alone never starts a
// goroutine.
func (worker *Worker) Start(
	parent context.Context,
) error {
	if worker == nil {
		return errors.New("nil queue worker")
	}
	if parent == nil {
		return errors.New("worker parent context required")
	}
	if err := parent.Err(); err != nil {
		return fmt.Errorf(
			"worker parent context already cancelled: %w",
			err,
		)
	}

	worker.mu.Lock()
	defer worker.mu.Unlock()

	if worker.started {
		return ErrWorkerAlreadyStarted
	}

	if worker.queue.Closed() {
		return ErrQueueClosed
	}

	ctx, cancel := context.WithCancel(parent)

	worker.started = true
	worker.running = true
	worker.cancel = cancel

	go worker.run(ctx)

	return nil
}

func (worker *Worker) run(ctx context.Context) {
	defer func() {
		worker.mu.Lock()
		worker.running = false
		close(worker.done)
		worker.mu.Unlock()
	}()

	for {
		job, err := worker.queue.Claim(
			ctx,
			ClaimRequest{
				Queue:  worker.queueName,
				Worker: worker.id,
			},
		)
		if err != nil {
			if errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded) ||
				errors.Is(err, ErrQueueClosed) {
				return
			}

			return
		}

		worker.execute(ctx, job)

		if ctx.Err() != nil {
			return
		}
	}
}

func (worker *Worker) execute(
	ctx context.Context,
	job Job,
) {
	result := make(chan error, 1)

	go func() {
		result <- callHandlerSafelyV1(
			ctx,
			worker.handler,
			job,
		)
	}()

	renewInterval := worker.queue.config.LeaseDuration / 2
	if renewInterval <= 0 {
		renewInterval = time.Millisecond
	}

	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	for {
		select {
		case handlerErr := <-result:
			if handlerErr == nil {
				_, _ = worker.queue.Ack(
					job.ID,
					worker.id,
				)
				return
			}

			_, _ = worker.queue.Nack(
				job.ID,
				worker.id,
				handlerErr,
			)
			return

		case <-ticker.C:
			_, err := worker.queue.RenewLease(
				job.ID,
				worker.id,
			)
			if err != nil {
				return
			}

		case <-ctx.Done():
			_, _ = worker.queue.Nack(
				job.ID,
				worker.id,
				ctx.Err(),
			)
			return
		}
	}
}

func callHandlerSafelyV1(
	ctx context.Context,
	handler Handler,
	job Job,
) (handlerErr error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		handlerErr = fmt.Errorf(
			"queue handler panic: %v",
			recovered,
		)
	}()

	return handler(ctx, job)
}

// Stop requests worker cancellation and waits for its run loop to finish.
// The queue remains available to other workers.
func (worker *Worker) Stop(
	ctx context.Context,
) error {
	if worker == nil {
		return errors.New("nil queue worker")
	}
	if ctx == nil {
		return errors.New("worker stop context required")
	}

	worker.mu.Lock()

	if !worker.started {
		worker.mu.Unlock()
		return ErrWorkerNotStarted
	}

	cancel := worker.cancel
	done := worker.done
	worker.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

// Wait waits for a started worker to exit without requesting cancellation.
func (worker *Worker) Wait(
	ctx context.Context,
) error {
	if worker == nil {
		return errors.New("nil queue worker")
	}
	if ctx == nil {
		return errors.New("worker wait context required")
	}

	worker.mu.Lock()

	if !worker.started {
		worker.mu.Unlock()
		return ErrWorkerNotStarted
	}

	done := worker.done
	worker.mu.Unlock()

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (worker *Worker) Running() bool {
	if worker == nil {
		return false
	}

	worker.mu.Lock()
	defer worker.mu.Unlock()

	return worker.running
}

func (worker *Worker) ID() string {
	if worker == nil {
		return ""
	}

	return worker.id
}

func (worker *Worker) QueueName() string {
	if worker == nil {
		return ""
	}

	return worker.queueName
}
