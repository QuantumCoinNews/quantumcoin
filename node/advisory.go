package node

import (
	"math"
	"os"
	"path/filepath"
	"quantumcoin/ai"
)

// Only canonical bootstrap installs this observer. Bare NewRuntime remains a
// side-effect-free adapter for tooling and tests. Startup reads but does not
// write, since the caller may subsequently select the API read-only mode.
func (r *Runtime) initializeAdvisory() {
	_, cfg := r.Snapshot()
	if cfg == nil || cfg.ChainFile == "" {
		return
	}
	dir := cfg.ChainFile + ".advisory"
	if absolute, err := filepath.Abs(dir); err == nil {
		dir = absolute
	}
	r.mu.Lock()
	if r.advisoryDir != "" {
		r.mu.Unlock()
		return
	}
	r.advisoryDir = dir
	r.mu.Unlock()
	if err := ai.LoadStateFrom(dir); err != nil {
		ai.RecordPersistenceFailure("load state", err)
	}
	r.RefreshAdvisoryTelemetry()
	ai.PushAlert(ai.AlertInfo, "NODE_STARTED", "canonical runtime initialized", "node")
	_ = r.AddObserver(ObserverFunc(func(event NodeEvent) {
		level := ai.AlertInfo
		if event.Severity == NodeEventSeverityWarning {
			level = ai.AlertWarning
		}
		if event.Severity == NodeEventSeverityError {
			level = ai.AlertError
		}
		ai.PushAlert(level, event.Type, event.Message, "node")
		r.RefreshAdvisoryTelemetry()
		r.persistAdvisory()
	}))
}

func (r *Runtime) RefreshAdvisoryTelemetry() {
	if r == nil {
		return
	}
	// No runtime-wide hash counter exists. Mark unavailable instead of
	// presenting a fabricated measurement of zero hashes/second.
	ai.UpdateTelemetry(r.PeerCount(), math.NaN())
}

func (r *Runtime) persistAdvisory() {
	if r == nil {
		return
	}
	r.mu.RLock()
	dir, readOnly := r.advisoryDir, r.APIReadOnly
	r.mu.RUnlock()
	if dir == "" || readOnly {
		return
	}
	r.RefreshAdvisoryTelemetry()
	if err := os.MkdirAll(dir, 0700); err != nil {
		ai.RecordPersistenceFailure("create state directory", err)
		return
	}
	if err := ai.SaveStateTo(dir); err != nil {
		ai.RecordPersistenceFailure("save state", err)
	}
}

func (r *Runtime) recordAdvisoryBlock(height int) {
	if r == nil || !ai.Enabled() {
		return
	}
	r.mu.RLock()
	dir, readOnly, chain := r.advisoryDir, r.APIReadOnly, r.Blockchain
	r.mu.RUnlock()
	if dir == "" || readOnly || chain == nil {
		return
	}
	reviews := ai.BuildSecurityReviews(chain.AIConfirmedBlockSample(height))
	for _, review := range reviews {
		ai.PushAlert(ai.AlertWarning, "TRANSACTION_REVIEW", review.Reason, "block-analysis")
	}
	if len(reviews) == 0 {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		ai.RecordPersistenceFailure("create review directory", err)
		return
	}
	if err := ai.WriteAISecurityReviewsTo(dir, reviews); err != nil {
		ai.RecordPersistenceFailure("security reviews", err)
	}
}
