package miner

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"
)

type refreshBackend struct {
	calls      atomic.Int32
	checks     atomic.Int32
	submit     atomic.Int32
	checked    chan struct{}
	submitting chan struct{}
	release    chan struct{}
	refreshed  chan struct{}
	deny       bool
}

func (b *refreshBackend) GetWork(ctx context.Context, address string) (*Work, error) {
	if b.calls.Add(1) == 2 && b.refreshed != nil {
		close(b.refreshed)
	}
	return &Work{Header: make([]byte, 16), Target: new(big.Int), Miner: address, ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (b *refreshBackend) WorkCurrent(ctx context.Context, w *Work) (bool, error) {
	if b.checks.Add(1) == 1 && b.checked != nil {
		close(b.checked)
	}
	if b.deny {
		return false, errNodeMiningDenied
	}
	return false, nil
}
func (b *refreshBackend) Submit(ctx context.Context, w *Work, n uint64, h string) (bool, error) {
	b.submit.Add(1)
	if b.submitting != nil {
		close(b.submitting)
	}
	if b.release != nil {
		select {
		case <-b.release:
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	return true, nil
}
func TestWorkerRefreshesUnsolvedTipAndStops(t *testing.T) {
	b := &refreshBackend{refreshed: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	w := NewWorker(b, WorkerConfig{Threads: 2, Address: "fixture", RefreshInterval: 5 * time.Millisecond})
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	select {
	case <-b.refreshed:
		cancel()
	case <-ctx.Done():
		t.Fatal("stale job was not replaced")
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if b.calls.Load() < 2 || b.submit.Load() != 0 {
		t.Fatal("unsolved stale work was not replaced", b.calls.Load(), b.submit.Load())
	}
}
func TestWorkerRefreshAuthorizationDenialStopsHashing(t *testing.T) {
	b := &refreshBackend{deny: true}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := NewWorker(b, WorkerConfig{Threads: 1, RefreshInterval: time.Millisecond}).Run(ctx)
	if !errors.Is(err, errNodeMiningDenied) || b.calls.Load() != 1 {
		t.Fatal(err, b.calls.Load())
	}
}
func TestWorkerTipChangeDoesNotDiscardInflightAcceptedReceipt(t *testing.T) {
	b := &refreshBackend{checked: make(chan struct{}), submitting: make(chan struct{}), release: make(chan struct{})}
	w := NewWorker(b, WorkerConfig{Threads: 2, RefreshInterval: time.Millisecond})
	work := &Work{Header: make([]byte, 16), Target: new(big.Int).Lsh(big.NewInt(1), 256), ExpiresAt: time.Now().Add(time.Minute)}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- w.solve(ctx, work) }()
	select {
	case <-b.submitting:
	case <-ctx.Done():
		t.Fatal("submit never began")
	}
	select {
	case <-b.checked:
	case <-ctx.Done():
		t.Fatal("refresh never checked")
	}
	close(b.release)
	if err := <-done; err != nil || w.Found() != 1 || b.submit.Load() != 1 {
		t.Fatal("receipt lost or duplicated", err, w.Found(), b.submit.Load())
	}
}
