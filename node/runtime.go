package node

import (
	"net/http"
	"sync"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/consensus"
	"quantumcoin/p2p"
)

const RuntimeVersionV1 = "QC_NODE_RUNTIME_V1"

// Runtime owns the long-lived node services and shared state.
//
// This first version is intentionally passive. Existing main.go behavior
// remains unchanged while services are moved behind this boundary in later
// runtime migration steps.
type Runtime struct {
	mu       sync.RWMutex
	miningMu sync.Mutex

	Blockchain    *blockchain.Blockchain
	Config        *config.Config
	p2pServer     *p2p.ServerV1
	HTTPServer    *http.Server
	minerStop     chan struct{}
	minerDone     chan struct{}
	minerAddress  string
	minerStopping bool
	observers     []Observer

	// Immutable signed game program; only canonical bootstrap installs it.
	nftGameProgram consensus.VerifiedNFTMonthlyProgramV1

	APIReadOnly bool

	ChainReloadMu       sync.Mutex
	ChainLastModifiedAt time.Time
	ChainLastSize       int64
}

// NewRuntime creates a passive node runtime around existing components.
func NewRuntime(
	bc *blockchain.Blockchain,
	cfg *config.Config,
) *Runtime {
	return &Runtime{
		Blockchain: bc,
		Config:     cfg,
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
