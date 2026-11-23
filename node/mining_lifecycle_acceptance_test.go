package node

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

func cancellableLegacyRuntimeV1(t *testing.T) *Runtime {
	t.Helper()
	chain := blockchain.NewIsolatedLocalBlockchainV1(50, 0)
	c := config.Default()
	c.ChainFile = filepath.Join(t.TempDir(), config.QCIsolatedLocalChainFileV1)
	if err := chain.SaveToFile(c.ChainFile); err != nil {
		t.Fatal(err)
	}
	r := NewRuntime(chain, c)
	t.Cleanup(func() {
		r.StopMining()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := r.WaitMiningContext(ctx); err != nil {
			t.Error(err)
		}
	})
	return r
}

func waitLegacySearchBusyV1(t *testing.T, r *Runtime) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		done := make(chan struct{})
		go func() { _ = r.MiningTip(); close(done) }()
		select {
		case <-done:
			time.Sleep(time.Millisecond)
		case <-time.After(20 * time.Millisecond):
			return
		}
	}
	t.Fatal("legacy search never held its chain snapshot")
}

func TestLegacyConcurrentStopUnblocksAutosaveAndRestart(t *testing.T) {
	r := cancellableLegacyRuntimeV1(t)
	var reports atomic.Int32
	hooks := MiningHooksV1{Difficulty: func() int { return 255 }, OnError: func(error) { reports.Add(1) }}
	if err := r.StartMining(wallet.NewWallet().GetAddress(), hooks); err != nil {
		t.Fatal(err)
	}
	waitLegacySearchBusyV1(t, r)
	saveDone := make(chan error, 1)
	go func() { saveDone <- r.SaveChain() }()
	var wg sync.WaitGroup
	var stopped atomic.Int32
	for range 32 {
		wg.Go(func() {
			if r.StopMining() {
				stopped.Add(1)
			}
		})
	}
	wg.Wait()
	if stopped.Load() != 1 {
		t.Fatal("stop ownership was not exclusive", stopped.Load())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.WaitMiningContext(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-saveDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("autosave remained blocked after cancel")
	}
	if reports.Load() != 0 {
		t.Fatal("normal cancellation emitted a mining error")
	}
	chain, c := r.Snapshot()
	loaded, err := blockchain.LoadBlockchainFromFile(c.ChainFile)
	if err != nil || loaded.BestHeightSnapshot() != 0 || !bytes.Equal(chain.BlockByIndexSnapshot(0).Hash, loaded.BlockByIndexSnapshot(0).Hash) {
		t.Fatal("stop/autosave damaged the chain", err)
	}
	if err := r.StartMining(wallet.NewWallet().GetAddress(), hooks); err != nil {
		t.Fatal("restart failed", err)
	}
	r.StopMining()
	if err := r.WaitMiningContext(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyShutdownCancelsActiveSearchBeforeTakingRuntimeLock(t *testing.T) {
	r := cancellableLegacyRuntimeV1(t)
	if err := r.StartMining(wallet.NewWallet().GetAddress(), MiningHooksV1{Difficulty: func() int { return 255 }}); err != nil {
		t.Fatal(err)
	}
	waitLegacySearchBusyV1(t, r)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.Shutdown(ctx); err != nil {
		t.Fatal("shutdown could not interrupt the locked search", err)
	}
	if r.IsMining() {
		t.Fatal("shutdown left a miner registered")
	}
	if err := r.StartMining(wallet.NewWallet().GetAddress(), MiningHooksV1{Difficulty: func() int { return 1 }}); !errors.Is(err, ErrRuntimeShuttingDownV1) {
		t.Fatal("start escaped shutdown", err)
	}
	_, c := r.Snapshot()
	if _, err := blockchain.LoadBlockchainFromFile(c.ChainFile); err != nil {
		t.Fatal("final saved chain is not loadable", err)
	}
}

func TestLateStopSignalCannotCancelReplacementRun(t *testing.T) {
	r := cancellableLegacyRuntimeV1(t)
	chain, _ := r.Snapshot()
	entered, release := make(chan struct{}), make(chan struct{})
	if err := r.StartMining(wallet.NewWallet().GetAddress(), MiningHooksV1{Difficulty: func() int { close(entered); <-release; return 1 }}); err != nil {
		t.Fatal(err)
	}
	<-entered
	old := r.activeMiner.Load()
	r.mu.Lock()
	r.Blockchain = nil
	r.mu.Unlock()
	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.WaitMiningContext(ctx); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	r.Blockchain = chain
	r.mu.Unlock()
	if err := r.StartMining(wallet.NewWallet().GetAddress(), MiningHooksV1{Difficulty: func() int { return 255 }}); err != nil {
		t.Fatal(err)
	}
	replacement := r.activeMiner.Load()
	if old == replacement || !old.requestStopV1() || replacement.requested.Load() || !r.IsMining() {
		t.Fatal("old signal affected the replacement run")
	}
	r.StopMining()
	if err := r.WaitMiningContext(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestMiningCancellationWhileQueuedForRuntime(t *testing.T) {
	r := cancellableLegacyRuntimeV1(t)
	r.miningMu.Lock()
	defer r.miningMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if block, err := r.MineOnceContextV1(ctx, wallet.NewWallet().GetAddress(), 1); block != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("queued runtime call did not cancel", err)
	}
}

func TestLegacyShutdownCancelsUnregisteredOneShot(t *testing.T) {
	r := cancellableLegacyRuntimeV1(t)
	result := make(chan error, 1)
	go func() {
		_, err := r.MineOnceContextV1(context.Background(), wallet.NewWallet().GetAddress(), 255)
		result <- err
	}()
	waitLegacySearchBusyV1(t, r)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := r.Shutdown(ctx); err != nil {
		t.Fatal("shutdown blocked on one-shot", err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("one-shot survived shutdown")
	}
}
