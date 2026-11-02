package node

import (
	"fmt"
	"os"
	"strings"

	"quantumcoin/blockchain"
)

const ChainReloadVersionV1 = "QC_NODE_CHAIN_RELOAD_V1"

// ReloadChainFromDiskIfChanged reloads the configured chain file only when
// the runtime is operating in API read-only mode and the file has changed.
//
// It returns true only when the runtime-owned blockchain was replaced.
func (r *Runtime) ReloadChainFromDiskIfChanged() (
	bool,
	error,
) {
	if r == nil {
		return false, fmt.Errorf(
			"%s: runtime is unavailable",
			ChainReloadVersionV1,
		)
	}

	r.mu.RLock()
	readOnly := r.APIReadOnly
	cfg := r.Config
	current := r.Blockchain
	r.mu.RUnlock()

	if !readOnly {
		return false, nil
	}

	if cfg == nil {
		return false, fmt.Errorf(
			"%s: config is unavailable",
			ChainReloadVersionV1,
		)
	}

	chainFile := strings.TrimSpace(
		cfg.ChainFile,
	)
	if chainFile == "" {
		return false, fmt.Errorf(
			"%s: chain file is empty",
			ChainReloadVersionV1,
		)
	}

	stat, err := os.Stat(chainFile)
	if err != nil {
		return false, fmt.Errorf(
			"%s: inspect chain file %q: %w",
			ChainReloadVersionV1,
			chainFile,
			err,
		)
	}

	r.ChainReloadMu.Lock()
	defer r.ChainReloadMu.Unlock()

	if !r.ChainLastModifiedAt.IsZero() &&
		r.ChainLastModifiedAt.Equal(
			stat.ModTime(),
		) &&
		r.ChainLastSize == stat.Size() {
		return false, nil
	}

	chain, err :=
		blockchain.LoadBlockchainFromFileWithLocalContextV1(
			chainFile,
			current,
		)
	if err != nil {
		return false, fmt.Errorf(
			"%s: load blockchain %q: %w",
			ChainReloadVersionV1,
			chainFile,
			err,
		)
	}

	// Trusted loading has already restored local policies and maturity.

	r.mu.Lock()
	if r.Blockchain != current {
		r.mu.Unlock()
		return false, fmt.Errorf("%s: runtime chain changed during trusted reload", ChainReloadVersionV1)
	}
	r.Blockchain = chain
	r.mu.Unlock()

	r.ChainLastModifiedAt = stat.ModTime()
	r.ChainLastSize = stat.Size()

	r.EmitEvent(
		NodeEvent{
			Type:     "CHAIN_RELOADED",
			Severity: NodeEventSeverityInfo,
			Message:  "runtime chain reloaded from disk",
			Data: map[string]any{
				"chainFile": chainFile,
				"height":    chain.BestHeightSnapshot(),
				"size":      stat.Size(),
			},
		},
	)

	return true, nil
}
