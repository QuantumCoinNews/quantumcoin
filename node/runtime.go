package node

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consensus"
	"quantumcoin/monitoring"
	"quantumcoin/p2p"
)

const RuntimeVersionV1 = "QC_NODE_RUNTIME_V1"

// Runtime owns the long-lived node services and shared state.
//
// Bootstrap installs the canonical chain and policies. Runtime owns HTTP, P2P,
// mining and graceful shutdown; command and HTTP adapters delegate to it.
type Runtime struct {
	mu                sync.RWMutex
	miningMu          sync.Mutex
	stopping          bool
	shutdownDone      chan struct{}
	shutdownOnce      sync.Once
	shutdownRequested atomic.Bool
	activeMiner       atomic.Pointer[miningStopSignalV1]

	Blockchain *blockchain.Blockchain
	Config     *config.Config
	p2pServer  *p2p.ServerV1
	// Peer handlers keep a chain reference for their connection lifetime.
	// Once P2P has used this runtime, pointer replacement requires a restart.
	p2pChainPinned     bool
	HTTPServer         *http.Server
	minerStop          chan struct{}
	minerDone          chan struct{}
	minerAddress       string
	observers          []Observer
	monitor            *monitoring.Engine
	monitorEpoch       string
	httpMonitorBudgets [3]httpObservationBudgetV1
	advisoryDir        string

	// Immutable signed game program; only canonical bootstrap installs it.
	nftGameProgram consensus.VerifiedNFTMonthlyProgramV1

	APIReadOnly bool

	ChainReloadMu       sync.Mutex
	ChainLastModifiedAt time.Time
	ChainLastSize       int64
}

// NewRuntime wraps existing components without starting services. Production
// callers must use Bootstrap or BootstrapWithConfig to enforce startup policy.
func NewRuntime(
	bc *blockchain.Blockchain,
	cfg *config.Config,
) *Runtime {
	return &Runtime{
		Blockchain:   bc,
		Config:       cfg,
		shutdownDone: make(chan struct{}),
	}
}

// Snapshot returns the currently owned shared components.
func (r *Runtime) Snapshot() (
	*blockchain.Blockchain,
	*config.Config,
) {
	if r == nil {
		return nil, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.Blockchain, r.Config
}

// SetHTTPServer records the active HTTP server.
func (r *Runtime) SetHTTPServer(server *http.Server) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.HTTPServer = server
}

// HTTP returns the active HTTP server.
func (r *Runtime) HTTP() *http.Server {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.HTTPServer
}

// SetAPIReadOnly changes the runtime API mode.
func (r *Runtime) SetAPIReadOnly(readOnly bool) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.APIReadOnly = readOnly
}

// IsAPIReadOnly reports whether the runtime API is read-only.
func (r *Runtime) IsAPIReadOnly() bool {
	if r == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.APIReadOnly
}
