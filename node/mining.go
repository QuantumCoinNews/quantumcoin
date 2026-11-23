package node

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"quantumcoin/blockchain"
)

const MiningRuntimeVersionV1 = "QC_NODE_MINING_RUNTIME_V1"

var ErrMinerAlreadyRunningV1 = errors.New("miner already running")
var ErrRuntimeReadOnlyV1 = errors.New("node runtime is read-only")
var ErrRuntimeShuttingDownV1 = errors.New("node runtime is shutting down")
var ErrMiningNetworkMismatchV1 = errors.New("mining requires the expected isolated local genesis")

var ErrMiningBlockchainUnavailableV1 = errors.New(
	"mining blockchain is unavailable",
)

type MiningHooksV1 struct {
	Difficulty func() int

	ErrorDelay func(time.Duration) time.Duration

	Broadcast func(*blockchain.Block)

	OnBlockMined func(*blockchain.Block)

	OnError func(error)
}

// MineOnceV1 is the canonical node-runtime entry point for producing one
// block. It serializes local mining callers while preserving the existing
// blockchain mining behavior.
func (r *Runtime) MineOnceV1(
	miner string,
	difficulty int,
) (*blockchain.Block, error) {
	return r.MineOnceContextV1(context.Background(), miner, difficulty)
}

// MineIsolatedLocalBlockV1 checks the pinned genesis under the same runtime
// lock used for mining. A reconnect to a different node cannot bypass it.
func (r *Runtime) MineIsolatedLocalBlockV1(miner string, difficulty int, genesisHash string) (*blockchain.Block, error) {
	if genesisHash == "" {
		return nil, ErrMiningNetworkMismatchV1
	}
	return r.mineOnceV1(miner, difficulty, genesisHash)
}

func (r *Runtime) mineOnceV1(
	miner string,
	difficulty int,
	localGenesis string,
) (result *blockchain.Block, err error) {
	ctx, cancel, err := r.miningContextV1(context.Background())
	if err != nil {
		return nil, err
	}
	defer cancel()
	return r.mineOnceContextV1(ctx, miner, difficulty, localGenesis)
}

func (r *Runtime) mineOnceContextV1(ctx context.Context, miner string, difficulty int, localGenesis string) (result *blockchain.Block, err error) {
	defer func() {
		if result != nil && err == nil {
			r.recordAdvisoryBlock(result.Index)
			r.persistAdvisory()
		}
	}()
	if r == nil {
		return nil, fmt.Errorf(
			"%s: %w",
			MiningRuntimeVersionV1,
			ErrMiningBlockchainUnavailableV1,
		)
	}

	if err := r.lockMiningContextV1(ctx); err != nil {
		return nil, err
	}
	defer r.miningMu.Unlock()

	r.mu.RLock()
	defer r.mu.RUnlock()
	chain := r.Blockchain
	if r.stopping {
		return nil, ErrRuntimeShuttingDownV1
	}
	if r.APIReadOnly {
		return nil, ErrRuntimeReadOnlyV1
	}

	if chain == nil {
		return nil, fmt.Errorf(
			"%s: %w",
			MiningRuntimeVersionV1,
			ErrMiningBlockchainUnavailableV1,
		)
	}
	if localGenesis != "" {
		id := chain.NetworkIdentitySnapshotV1()
		if id == nil || id.Network != "local-release" ||
			id.GenesisDomain != blockchain.IsolatedLocalGenesisDomainV1 || id.GenesisHash != localGenesis {
			return nil, ErrMiningNetworkMismatchV1
		}
		if r.Config == nil {
			return nil, ErrMiningBlockchainUnavailableV1
		}
	}
	if r.Config != nil && strings.TrimSpace(r.Config.ChainFile) != "" {
		return chain.MineBlockAndPersistContextV1(ctx, miner, difficulty, r.Config.ChainFile)
	}
	if localGenesis != "" {
		return nil, ErrMiningBlockchainUnavailableV1
	}

	return chain.MineBlockContextV1(
		ctx, miner,
		difficulty,
	)
}

func (r *Runtime) StartMining(
	miner string,
	hooks MiningHooksV1,
) error {
	if r == nil {
		return fmt.Errorf(
			"%s: runtime is unavailable",
			MiningRuntimeVersionV1,
		)
	}

	miner = strings.TrimSpace(miner)
	if miner == "" {
		return fmt.Errorf(
			"%s: miner address is empty",
			MiningRuntimeVersionV1,
		)
	}

	if hooks.Difficulty == nil {
		return fmt.Errorf(
			"%s: difficulty provider is unavailable",
			MiningRuntimeVersionV1,
		)
	}
	if r.shutdownRequested.Load() {
		return ErrRuntimeShuttingDownV1
	}
	if r.activeMiner.Load() != nil {
		return ErrMinerAlreadyRunningV1
	}

	r.mu.Lock()
	if r.stopping || r.shutdownRequested.Load() {
		r.mu.Unlock()
		return ErrRuntimeShuttingDownV1
	}
	if r.APIReadOnly {
		r.mu.Unlock()
		return ErrRuntimeReadOnlyV1
	}

	if r.Blockchain == nil {
		r.mu.Unlock()

		return fmt.Errorf(
			"%s: blockchain is unavailable",
			MiningRuntimeVersionV1,
		)
	}

	if r.minerStop != nil {
		r.mu.Unlock()

		return ErrMinerAlreadyRunningV1
	}

	stop := make(chan struct{})
	done := make(chan struct{})

	r.minerStop = stop
	r.minerDone = done
	r.minerAddress = miner
	r.activeMiner.Store(&miningStopSignalV1{stop: stop})

	r.mu.Unlock()

	r.EmitEvent(
		NodeEvent{
			Type:     "MINER_STARTED",
			Severity: NodeEventSeverityInfo,
			Message:  "continuous miner started",
			Data: map[string]any{
				"miner": miner,
			},
		},
	)

	go r.runMiningLoopV1(
		miner,
		stop,
		done,
		hooks,
	)

	return nil
}

func (r *Runtime) runMiningLoopV1(
	miner string,
	stop <-chan struct{},
	done chan<- struct{},
	hooks MiningHooksV1,
) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-stop:
			cancel()
		case <-ctx.Done():
		}
	}()
	defer func() {
		r.mu.Lock()

		if r.minerStop == stop {
			r.minerStop = nil
			r.minerDone = nil
			r.minerAddress = ""
			if signal := r.activeMiner.Load(); signal != nil && signal.stop == stop {
				r.activeMiner.CompareAndSwap(signal, nil)
			}
		}

		// Completion means service state has also been cleared. Shutdown and
		// a replacement miner must never observe "done" with stale ownership.
		close(done)
		r.mu.Unlock()

		r.EmitEvent(
			NodeEvent{
				Type:     "MINER_STOPPED",
				Severity: NodeEventSeverityInfo,
				Message:  "continuous miner stopped",
				Data: map[string]any{
					"miner": miner,
				},
			},
		)
	}()

	for {
		select {
		case <-stop:
			return

		default:
		}

		difficulty := hooks.Difficulty()

		block, err := r.MineOnceContextV1(
			ctx, miner,
			difficulty,
		)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, ErrRuntimeShuttingDownV1) {
				return
			}
			r.reportMiningErrorV1(
				err,
				hooks,
			)

			if errors.Is(err, ErrRuntimeShuttingDownV1) || errors.Is(err, ErrRuntimeReadOnlyV1) || errors.Is(
				err,
				ErrMiningBlockchainUnavailableV1,
			) {
				return
			}

			delay := 1200 * time.Millisecond

			if hooks.ErrorDelay != nil {
				delay = hooks.ErrorDelay(
					delay,
				)
			}

			if delay > 0 {
				select {
				case <-stop:
					return

				case <-time.After(delay):
				}
			}

			continue
		}

		if hooks.Broadcast != nil {
			hooks.Broadcast(block)
		}

		r.EmitEvent(
			NodeEvent{
				Type:     "BLOCK_MINED",
				Severity: NodeEventSeverityInfo,
				Message:  "block mined",
				Data: map[string]any{
					"miner":      miner,
					"height":     block.Index,
					"difficulty": difficulty,
				},
			},
		)

		if hooks.OnBlockMined != nil {
			hooks.OnBlockMined(block)
		}

		select {
		case <-stop:
			return

		case <-time.After(
			50 * time.Millisecond,
		):
		}
	}
}

func (r *Runtime) reportMiningErrorV1(
	err error,
	hooks MiningHooksV1,
) {
	if err == nil {
		return
	}

	r.EmitEvent(
		NodeEvent{
			Type:     "MINER_FAILED",
			Severity: NodeEventSeverityError,
			Message:  err.Error(),
		},
	)

	if hooks.OnError != nil {
		hooks.OnError(err)
	}
}

func (r *Runtime) StopMining() bool {
	if r == nil {
		return false
	}

	signal := r.activeMiner.Load()
	if !signal.requestStopV1() {
		return false
	}

	// The channel remains registered until the mining goroutine exits.
	// This prevents duplicate close and prevents another miner starting
	// while the previous MineBlock operation is still completing.
	return true
}

func (r *Runtime) IsMining() bool {
	if r == nil {
		return false
	}

	return r.activeMiner.Load() != nil
}

func (r *Runtime) MinerAddress() string {
	if r == nil {
		return ""
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.minerAddress
}

// WaitMining blocks until the active mining goroutine exits.
// It returns immediately when no miner is active.
func (r *Runtime) WaitMining() {
	if r == nil {
		return
	}

	r.mu.RLock()
	done := r.minerDone
	r.mu.RUnlock()

	if done == nil {
		return
	}

	<-done
}

// WaitMiningContext waits until the active miner exits or the context ends.
// It returns nil immediately when no miner is active.
func (r *Runtime) WaitMiningContext(
	ctx context.Context,
) error {
	if r == nil {
		return nil
	}

	if ctx == nil {
		return fmt.Errorf(
			"%s: wait mining context is unavailable",
			MiningRuntimeVersionV1,
		)
	}

	r.mu.RLock()
	done := r.minerDone
	r.mu.RUnlock()

	if done == nil {
		return nil
	}

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		return fmt.Errorf(
			"%s: wait miner shutdown: %w",
			MiningRuntimeVersionV1,
			ctx.Err(),
		)
	}
}
