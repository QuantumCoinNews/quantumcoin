package node

import (
	"context"
	"errors"
	"fmt"
)

const ShutdownVersionV1 = "QC_NODE_SHUTDOWN_V1"

// Shutdown gracefully stops runtime-owned services and persists the chain.
//
// Runtime owns shutdown ordering:
//  1. request miner stop
//  2. wait for miner exit
//  3. stop HTTP service
//  4. persist final chain state
//
// Final chain persistence is skipped if the miner cannot be confirmed stopped,
// preventing a shutdown save from racing an active mining mutation.
func (r *Runtime) Shutdown(
	ctx context.Context,
) error {
	if r == nil {
		return fmt.Errorf(
			"%s: runtime is unavailable",
			ShutdownVersionV1,
		)
	}

	if ctx == nil {
		return fmt.Errorf(
			"%s: context is unavailable",
			ShutdownVersionV1,
		)
	}

	r.EmitEvent(
		NodeEvent{
			Type:     "NODE_SHUTDOWN_STARTED",
			Severity: NodeEventSeverityInfo,
			Message:  "node shutdown started",
		},
	)

	var shutdownErr error
	minerStopped := true

	if r.IsMining() {
		r.StopMining()

		if err := r.WaitMiningContext(ctx); err != nil {
			minerStopped = false

			shutdownErr = errors.Join(
				shutdownErr,
				fmt.Errorf(
					"%s: miner shutdown: %w",
					ShutdownVersionV1,
					err,
				),
			)
		}
	}

	if err := r.StopP2P(); err != nil {
		shutdownErr = errors.Join(
			shutdownErr,
			fmt.Errorf(
				"%s: P2P shutdown: %w",
				ShutdownVersionV1,
				err,
			),
		)
	}

	server := r.HTTP()

	if server != nil {
		if err := server.Shutdown(ctx); err != nil {
			shutdownErr = errors.Join(
				shutdownErr,
				fmt.Errorf(
					"%s: HTTP shutdown: %w",
					ShutdownVersionV1,
					err,
				),
			)
		}
	}

	if minerStopped {
		if err := r.SaveChain(); err != nil {
			shutdownErr = errors.Join(
				shutdownErr,
				fmt.Errorf(
					"%s: final chain save: %w",
					ShutdownVersionV1,
					err,
				),
			)
		}
	}

	if shutdownErr != nil {
		r.EmitEvent(
			NodeEvent{
				Type:     "NODE_SHUTDOWN_FAILED",
				Severity: NodeEventSeverityError,
				Message:  shutdownErr.Error(),
			},
		)

		return shutdownErr
	}

	r.EmitEvent(
		NodeEvent{
			Type:     "NODE_SHUTDOWN_COMPLETED",
			Severity: NodeEventSeverityInfo,
			Message:  "node shutdown completed",
		},
	)

	return nil
}
