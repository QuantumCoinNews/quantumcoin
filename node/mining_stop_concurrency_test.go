package node

import (
	"context"
	"errors"
	"quantumcoin/blockchain"
	"quantumcoin/wallet"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentStopKeepsOldLoopBoundToItsOwnChannel(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	cfg, chain, _ := mandatoryStartupFixtureV1(t)
	r := NewRuntime(chain, cfg)
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	hooks := MiningHooksV1{Difficulty: func() int { once.Do(func() { close(entered); <-release }); return 8 }}
	if err := r.StartMining(wallet.NewWallet().GetAddress(), hooks); err != nil {
		t.Fatal(err)
	}
	<-entered
	var wg sync.WaitGroup
	var stopped atomic.Int32
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r.StopMining() {
				stopped.Add(1)
			}
		}()
	}
	wg.Wait()
	if stopped.Load() != 1 {
		t.Fatal("stop ownership", stopped.Load())
	}
	if err := r.StartMining(wallet.NewWallet().GetAddress(), hooks); !errors.Is(err, ErrMinerAlreadyRunningV1) {
		t.Fatal("restart before old loop completed", err)
	}
	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.WaitMiningContext(ctx); err != nil {
		t.Fatal(err)
	}
	if r.IsMining() {
		t.Fatal("old loop still active")
	}
	// A second run gets a different channel; old stop callers cannot close it.
	entered = make(chan struct{})
	release = make(chan struct{})
	once = sync.Once{}
	if err := r.StartMining(wallet.NewWallet().GetAddress(), hooks); err != nil {
		t.Fatal(err)
	}
	<-entered
	var shuts sync.WaitGroup
	shutdownErrors := make(chan error, 8)
	for i := 0; i < 8; i++ {
		shuts.Add(1)
		go func() { defer shuts.Done(); shutdownErrors <- r.Shutdown(ctx) }()
	}
	r.StopMining()
	close(release)
	shuts.Wait()
	close(shutdownErrors)
	for err := range shutdownErrors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if r.IsMining() {
		t.Fatal("shutdown left active loop")
	}
	if _, err := r.MineOnceV1(wallet.NewWallet().GetAddress(), 8); !errors.Is(err, ErrRuntimeShuttingDownV1) {
		t.Fatal("mining after shutdown", err)
	}
}

func TestExternalMiningRuntimeRejectsObserverAndShutdown(t *testing.T) {
	cfg, chain, _ := mandatoryStartupFixtureV1(t)
	r := NewRuntime(chain, cfg)
	r.SetAPIReadOnly(true)
	if _, _, err := r.PrepareExternalMining(wallet.NewWallet().GetAddress()); !errors.Is(err, ErrRuntimeReadOnlyV1) {
		t.Fatal(err)
	}
	if _, err := r.CommitExternalMining(&blockchain.Block{}, 0); !errors.Is(err, ErrRuntimeReadOnlyV1) {
		t.Fatal(err)
	}
	r.SetAPIReadOnly(false)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.PrepareExternalMining(wallet.NewWallet().GetAddress()); !errors.Is(err, ErrRuntimeShuttingDownV1) {
		t.Fatal(err)
	}
	if _, err := r.CommitExternalMining(&blockchain.Block{}, 0); !errors.Is(err, ErrRuntimeShuttingDownV1) {
		t.Fatal(err)
	}
}
