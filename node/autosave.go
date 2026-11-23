package node

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const AutosaveVersionV1 = "QC_NODE_AUTOSAVE_V1"

// SaveChain persists the runtime-owned blockchain to its configured chain file.
func (r *Runtime) SaveChain() error {
	if r == nil {
		return fmt.Errorf(
			"%s: runtime is unavailable",
			AutosaveVersionV1,
		)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	chain := r.Blockchain
	cfg := r.Config
	if r.APIReadOnly {
		return nil
	}

	if chain == nil {
		return fmt.Errorf(
			"%s: blockchain is unavailable",
			AutosaveVersionV1,
		)
	}

	if cfg == nil {
		return fmt.Errorf(
			"%s: config is unavailable",
			AutosaveVersionV1,
		)
	}

	chainFile := strings.TrimSpace(
		cfg.ChainFile,
	)
	if chainFile == "" {
		return fmt.Errorf(
			"%s: chain file is empty",
			AutosaveVersionV1,
		)
	}

	if err := chain.SaveToFile(chainFile); err != nil {
		return fmt.Errorf(
			"%s: save chain %q: %w",
			AutosaveVersionV1,
			chainFile,
			err,
		)
	}

	return nil
}

// RunAutosave periodically persists the runtime-owned blockchain until the
// supplied context is cancelled.
func (r *Runtime) RunAutosave(
	ctx context.Context,
	interval time.Duration,
	onError func(error),
) error {
	if r == nil {
		return fmt.Errorf(
			"%s: runtime is unavailable",
			AutosaveVersionV1,
		)
	}

	if ctx == nil {
		return fmt.Errorf(
			"%s: context is unavailable",
			AutosaveVersionV1,
		)
	}

	if interval <= 0 {
		return fmt.Errorf(
			"%s: interval must be positive",
			AutosaveVersionV1,
		)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	r.mu.RLock()
	shutdownDone := r.shutdownDone
	r.mu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-shutdownDone:
			return nil

		case <-ticker.C:
			r.PollMonitoring()
			r.persistAdvisory()
			if err := r.SaveChain(); err != nil &&
				onError != nil {
				onError(err)
			}
		}
	}
}
