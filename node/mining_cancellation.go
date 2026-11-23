package node

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// The stop signal is immutable per run. Cancellation never waits for runtime
// or chain locks held by the search being canceled.
type miningStopSignalV1 struct {
	stop      chan struct{}
	requested atomic.Bool
}

func (s *miningStopSignalV1) requestStopV1() bool {
	if s == nil || !s.requested.CompareAndSwap(false, true) {
		return false
	}
	close(s.stop)
	return true
}

func (r *Runtime) miningContextV1(parent context.Context) (context.Context, context.CancelFunc, error) {
	if r == nil {
		return nil, nil, ErrMiningBlockchainUnavailableV1
	}
	if parent == nil {
		return nil, nil, fmt.Errorf("mining context is unavailable")
	}
	if r.shutdownRequested.Load() {
		return nil, nil, ErrRuntimeShuttingDownV1
	}
	ctx, cancel := context.WithCancel(parent)
	done := r.shutdownDone // Initialized once by NewRuntime, never reassigned.
	go func() {
		if done != nil {
			select {
			case <-done:
				cancel()
			case <-ctx.Done():
			}
			return
		}
		// Keep manually constructed compatibility runtimes cancellable too.
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if r.shutdownRequested.Load() {
					cancel()
					return
				}
			}
		}
	}()
	return ctx, cancel, nil
}

func (r *Runtime) lockMiningContextV1(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if r.miningMu.TryLock() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}
