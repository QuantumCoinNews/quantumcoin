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
	if r == nil {
		return nil, fmt.Errorf(
			"%s: %w",
			MiningRuntimeVersionV1,
			ErrMiningBlockchainUnavailableV1,
		)
	}

	r.miningMu.Lock()
	defer r.miningMu.Unlock()

	r.mu.RLock()
	chain := r.Blockchain
	r.mu.RUnlock()

	if chain == nil {
		return nil, fmt.Errorf(
			"%s: %w",
			MiningRuntimeVersionV1,
			ErrMiningBlockchainUnavailableV1,
		)
	}

	return chain.MineBlock(
		miner,
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

	r.mu.Lock()

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
	defer func() {
		close(done)

		r.mu.Lock()

		if r.minerStop == stop {
			r.minerStop = nil
			r.minerDone = nil
			r.minerAddress = ""
			r.minerStopping = false
		}

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

		block, err := r.MineOnceV1(
			miner,
			difficulty,
		)
		if err != nil {
			r.reportMiningErrorV1(
				err,
				hooks,
			)

			if errors.Is(
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

		if err := r.SaveChain(); err != nil {
			r.reportMiningErrorV1(
				err,
				hooks,
			)
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

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.minerStop == nil ||
		r.minerStopping {
		return false
	}

	r.minerStopping = true
	close(r.minerStop)

	// The channel remains registered until the mining goroutine exits.
	// This prevents duplicate close and prevents another miner starting
	// while the previous MineBlock operation is still completing.
	return true
}

func (r *Runtime) IsMining() bool {
	if r == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.minerStop != nil
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
