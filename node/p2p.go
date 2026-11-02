package node

import (
	"errors"
	"fmt"
	"strings"

	"quantumcoin/p2p"
)

const P2PRuntimeVersionV1 = "QC_NODE_P2P_RUNTIME_V1"

var ErrP2PRuntimeAlreadyRunningV1 = errors.New("node p2p runtime already running")

// ServeP2P starts the runtime-owned inbound P2P listener and blocks until
// the listener is stopped or an unrecoverable network error occurs.
func (r *Runtime) ServeP2P(
	addr string,
) error {
	if r == nil {
		return fmt.Errorf(
			"%s: runtime is unavailable",
			P2PRuntimeVersionV1,
		)
	}

	r.mu.Lock()

	if r.Blockchain == nil {
		r.mu.Unlock()

		return fmt.Errorf(
			"%s: blockchain is unavailable",
			P2PRuntimeVersionV1,
		)
	}

	if r.p2pServer != nil {
		r.mu.Unlock()

		return ErrP2PRuntimeAlreadyRunningV1
	}

	server, err :=
		p2p.NewServerV1(r.Blockchain)
	if err != nil {
		r.mu.Unlock()

		return fmt.Errorf(
			"%s: create server: %w",
			P2PRuntimeVersionV1,
			err,
		)
	}

	r.p2pServer = server

	r.mu.Unlock()

	defer func() {
		r.mu.Lock()

		if r.p2pServer == server {
			r.p2pServer = nil
		}

		r.mu.Unlock()
	}()

	r.EmitEvent(
		NodeEvent{
			Type:     "P2P_SERVER_STARTING",
			Severity: NodeEventSeverityInfo,
			Message:  "inbound p2p server starting",
			Data: map[string]any{
				"addr": addr,
			},
		},
	)

	if err := server.Serve(addr); err != nil {
		r.EmitEvent(
			NodeEvent{
				Type:     "P2P_SERVER_FAILED",
				Severity: NodeEventSeverityError,
				Message:  err.Error(),
				Data: map[string]any{
					"addr": addr,
				},
			},
		)

		return fmt.Errorf(
			"%s: serve: %w",
			P2PRuntimeVersionV1,
			err,
		)
	}

	r.EmitEvent(
		NodeEvent{
			Type:     "P2P_SERVER_STOPPED",
			Severity: NodeEventSeverityInfo,
			Message:  "inbound p2p server stopped",
		},
	)

	return nil
}

// StopP2P stops the runtime-owned inbound listener.
func (r *Runtime) StopP2P() error {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	server := r.p2pServer
	r.mu.RUnlock()

	var stopErr error

	if server != nil {
		if err := server.Close(); err != nil {
			stopErr = fmt.Errorf(
				"%s: stop server: %w",
				P2PRuntimeVersionV1,
				err,
			)
		}
	}

	closedPeers := p2p.CloseAllPeersV1()

	r.EmitEvent(
		NodeEvent{
			Type:     "P2P_NETWORK_STOP_REQUESTED",
			Severity: NodeEventSeverityInfo,
			Message:  "p2p listener and peer connections stop requested",
			Data: map[string]any{
				"closedPeers": closedPeers,
			},
		},
	)

	return stopErr
}

func (r *Runtime) IsP2PRunning() bool {
	if r == nil {
		return false
	}

	r.mu.RLock()
	server := r.p2pServer
	r.mu.RUnlock()

	return server != nil &&
		server.IsRunning()
}

func (r *Runtime) P2PListenAddress() string {
	if r == nil {
		return ""
	}

	r.mu.RLock()
	server := r.p2pServer
	r.mu.RUnlock()

	if server == nil {
		return ""
	}

	return server.ListenAddress()
}

func (r *Runtime) BroadcastP2P(
	msg p2p.Message,
) {
	if r == nil {
		return
	}

	p2p.BroadcastMessage(msg)
}

func (r *Runtime) PeerCount() int {
	if r == nil {
		return 0
	}

	return p2p.GetPeerCount()
}

// ConnectPeer creates an outbound P2P connection.
// Inbound listener ownership remains separate under ServeP2P.
func (r *Runtime) ConnectPeer(
	port string,
	address string,
) error {
	if r == nil {
		return fmt.Errorf(
			"%s: runtime is unavailable",
			P2PRuntimeVersionV1,
		)
	}

	r.mu.RLock()
	chain := r.Blockchain
	r.mu.RUnlock()

	if chain == nil {
		return fmt.Errorf(
			"%s: blockchain is unavailable",
			P2PRuntimeVersionV1,
		)
	}

	address = strings.TrimSpace(address)
	if address == "" {
		return fmt.Errorf(
			"%s: peer address is empty",
			P2PRuntimeVersionV1,
		)
	}

	r.EmitEvent(
		NodeEvent{
			Type:     "P2P_OUTBOUND_CONNECT_REQUESTED",
			Severity: NodeEventSeverityInfo,
			Message:  "outbound p2p connection requested",
			Data: map[string]any{
				"address": address,
			},
		},
	)

	p2p.ConnectToPeer(
		port,
		address,
		chain,
	)

	return nil
}

// PeerSecurityAvailable reports whether the canonical P2P security engine
// is available. This is read-only runtime visibility.
func (r *Runtime) PeerSecurityAvailable() bool {
	if r == nil {
		return false
	}

	return p2p.DefaultPeerSecurity != nil
}

// PeerReputationSnapshot returns the current read-only peer reputation view.
func (r *Runtime) PeerReputationSnapshot() map[string]any {
	if r == nil ||
		p2p.DefaultPeerSecurity == nil {
		return map[string]any{}
	}

	return p2p.DefaultPeerSecurity.ReputationSnapshot()
}
