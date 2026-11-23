package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func isolateNativeLifecycleV1(t *testing.T) {
	t.Helper()
	guiMining.Lock()
	oldCancel, oldDone, oldNotified := guiMining.cancel, guiMining.done, guiMining.notified
	if oldCancel != nil {
		guiMining.Unlock()
		t.Fatal("test started with an active worker")
	}
	guiMining.cancel = nil
	guiMining.done = nil
	guiMining.notified = nil
	guiMining.Unlock()
	oldState, oldUpdate := minerRunningState, onMinerStateUpdate
	t.Cleanup(func() {
		guiMining.Lock()
		if guiMining.cancel != nil {
			guiMining.cancel()
		}
		guiMining.cancel = oldCancel
		guiMining.done = oldDone
		guiMining.notified = oldNotified
		guiMining.Unlock()
		minerRunningState = oldState
		onMinerStateUpdate = oldUpdate
	})
}

func TestNativeMinerNaturalExitAndColdRestart(t *testing.T) {
	isolateNativeLifecycleV1(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	if !installNativeMiningRunV1(cancel, done) || !nativeMinerAlive() {
		t.Fatal("native run not registered")
	}
	if err := startNativeMining(); !errors.Is(err, errNativeMinerAlreadyRunningV1) {
		t.Fatal("busy native start reported success", err)
	}
	if installNativeMiningRunV1(cancel, make(chan struct{})) {
		t.Fatal("replacement admitted before completion")
	}
	if !completeNativeMiningRunV1(done) || nativeMinerAlive() {
		t.Fatal("natural exit retained restart lock")
	}
	select {
	case <-done:
	default:
		t.Fatal("completion not published before restart")
	}
	if completeNativeMiningRunV1(done) {
		t.Fatal("duplicate completion claimed ownership")
	}
	newer := make(chan struct{})
	if !installNativeMiningRunV1(cancel, newer) {
		t.Fatal("cold restart rejected")
	}
	if completeNativeMiningRunV1(done) {
		t.Fatal("old completion cleared replacement")
	}
	if !nativeMinerAlive() {
		t.Fatal("replacement lost ownership")
	}
	if ctx.Err() != nil {
		t.Fatal("registration canceled the owner's context")
	}
	completeNativeMiningRunV1(newer)
}

func TestNativeMinerQueuedStopNotificationCannotClearReplacement(t *testing.T) {
	isolateNativeLifecycleV1(t)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	old := make(chan struct{})
	installNativeMiningRunV1(cancel, old)
	completeNativeMiningRunV1(old)
	var queue []func()
	dispatch := func(fn func()) { queue = append(queue, fn) }
	var updates int
	onMinerStateUpdate = func(running bool) {
		if running {
			t.Error("unexpected running notification")
		}
		updates++
	}
	notifyNativeMiningStoppedV1(old, dispatch)
	current := make(chan struct{})
	installNativeMiningRunV1(cancel, current)
	minerRunningState = true
	queue[0]()
	if !minerRunningState || updates != 0 {
		t.Fatal("stale stop callback affected replacement")
	}
	completeNativeMiningRunV1(current)
	notifyNativeMiningStoppedV1(current, dispatch)
	notifyNativeMiningStoppedV1(current, dispatch)
	queue[1]()
	queue[2]()
	if minerRunningState || updates != 1 {
		t.Fatal("current completion was not reported exactly once")
	}
}

func TestNativeMinerConcurrentStopClosesOnlyOwnedRun(t *testing.T) {
	isolateNativeLifecycleV1(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	installNativeMiningRunV1(cancel, done)
	var reports atomic.Int32
	onMinerStateUpdate = func(bool) { reports.Add(1) }
	completion := make(chan struct{})
	go func() { <-ctx.Done(); completeNativeMiningRunV1(done); close(completion) }()
	immediate := func(fn func()) { fn() }
	var wg sync.WaitGroup
	failures := make(chan error, 16)
	for range 16 {
		wg.Go(func() { failures <- stopNativeMiningWithDispatchV1(immediate) })
	}
	wg.Wait()
	<-completion
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	if nativeMinerAlive() || reports.Load() != 1 {
		t.Fatal("concurrent stop retained worker or duplicated notification")
	}
}

func TestNativeMinerLateStopCannotClearNewRun(t *testing.T) {
	isolateNativeLifecycleV1(t)
	entered, release, old := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	installNativeMiningRunV1(func() { close(entered); <-release }, old)
	result := make(chan error, 1)
	go func() { result <- stopNativeMiningWithDispatchV1(func(fn func()) { fn() }) }()
	<-entered
	completeNativeMiningRunV1(old)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	current := make(chan struct{})
	installNativeMiningRunV1(cancel, current)
	minerRunningState = true
	onMinerStateUpdate = func(bool) { t.Error("stale stop emitted a state update") }
	unblock()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("old stop did not return")
	}
	if !nativeMinerAlive() || !minerRunningState {
		t.Fatal("old stop cleared the current worker")
	}
	completeNativeMiningRunV1(current)
}

func TestNativeMinerPendingStopRetainsOwnership(t *testing.T) {
	isolateNativeLifecycleV1(t)
	done := make(chan struct{})
	installNativeMiningRunV1(func() {}, done)
	minerRunningState = true
	updates := 0
	onMinerStateUpdate = func(bool) { updates++ }
	if err := stopNativeMiningWithDispatchV1(func(fn func()) { fn() }); err == nil {
		t.Fatal("unfinished stop reported success")
	}
	if !nativeMinerAlive() || !minerRunningState || updates != 0 {
		t.Fatal("pending stop cleared live ownership or UI")
	}
	completeNativeMiningRunV1(done)
	notifyNativeMiningStoppedV1(done, func(fn func()) { fn() })
	if nativeMinerAlive() || minerRunningState || updates != 1 {
		t.Fatal("eventual completion did not clear ownership")
	}
}

func TestNativeMinerFinalUIQueueRendersCurrentWorker(t *testing.T) {
	isolateNativeLifecycleV1(t)
	var queue []func()
	dispatch := func(fn func()) { queue = append(queue, fn) }
	var rendered []bool
	onMinerStateUpdate = nativeMiningUIStateHandlerV1(dispatch, func(running bool) {
		rendered = append(rendered, running)
	})
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	old := make(chan struct{})
	installNativeMiningRunV1(cancel, old)
	completeNativeMiningRunV1(old)
	// The owner guard passes, then the tab queues its final widget update.
	notifyNativeMiningStoppedV1(old, func(fn func()) { fn() })
	current := make(chan struct{})
	installNativeMiningRunV1(cancel, current)
	minerRunningState = true
	queue[0]()
	if !minerRunningState || len(rendered) != 1 || !rendered[0] {
		t.Fatal("queued stopped render hid the replacement worker", rendered)
	}
	// A queued start update must also reflect an intervening natural exit.
	onMinerStateUpdate(true)
	completeNativeMiningRunV1(current)
	queue[1]()
	if minerRunningState || len(rendered) != 2 || rendered[1] {
		t.Fatal("queued running render revived a completed worker", rendered)
	}
}
