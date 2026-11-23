package main

import "sync"

// A listener can return as soon as shutdown closes it, while active requests
// and the final save are still draining. Keep the CLI alive until its signal
// worker finishes. Normal startup failures cancel an unused signal worker.
type cliShutdownBarrierV1 struct {
	mu                sync.Mutex
	started, canceled bool
	stop, done        chan struct{}
}

func newCLIShutdownBarrierV1() *cliShutdownBarrierV1 {
	return &cliShutdownBarrierV1{stop: make(chan struct{}), done: make(chan struct{})}
}

func (b *cliShutdownBarrierV1) Begin() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.started || b.canceled {
		return false
	}
	b.started = true
	return true
}

func (b *cliShutdownBarrierV1) Finish() { close(b.done) }

func (b *cliShutdownBarrierV1) Wait() {
	b.mu.Lock()
	if !b.started {
		if !b.canceled {
			b.canceled = true
			close(b.stop)
		}
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()
	<-b.done
}
