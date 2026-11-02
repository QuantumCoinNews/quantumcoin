package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"quantumcoin/blockchain"
)

const P2PServerVersionV1 = "QC_P2P_SERVER_V1"

var ErrP2PServerAlreadyRunningV1 = errors.New("p2p server already running")

// ServerV1 owns the inbound P2P listener lifecycle.
//
// Existing peer maps, handshake validation and PeerSecurity remain in the
// p2p package. This type only introduces explicit listener ownership.
type ServerV1 struct {
	mu sync.RWMutex

	blockchain *blockchain.Blockchain
	listener   net.Listener
	listenAddr string
}

// NewServerV1 creates an inbound P2P server.
func NewServerV1(
	bc *blockchain.Blockchain,
) (*ServerV1, error) {
	if bc == nil {
		return nil, fmt.Errorf(
			"%s: blockchain is unavailable",
			P2PServerVersionV1,
		)
	}

	return &ServerV1{
		blockchain: bc,
	}, nil
}

func normalizeListenAddressV1(
	addr string,
) (string, error) {
	addr = strings.TrimSpace(addr)

	if addr == "" {
		return "", fmt.Errorf(
			"%s: listen address is empty",
			P2PServerVersionV1,
		)
	}

	// Plain numeric port: "8333" -> ":8333".
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	return addr, nil
}

// Serve opens the inbound listener and blocks until the server is closed
// or an unrecoverable Accept error occurs.
func (s *ServerV1) Serve(
	addr string,
) error {
	if s == nil {
		return fmt.Errorf(
			"%s: server is unavailable",
			P2PServerVersionV1,
		)
	}

	addr, err :=
		normalizeListenAddressV1(addr)
	if err != nil {
		return err
	}

	s.mu.Lock()

	if s.listener != nil {
		s.mu.Unlock()

		return ErrP2PServerAlreadyRunningV1
	}

	listener, err := net.Listen(
		"tcp",
		addr,
	)
	if err != nil {
		s.mu.Unlock()

		return fmt.Errorf(
			"%s: listen %q: %w",
			P2PServerVersionV1,
			addr,
			err,
		)
	}

	s.listener = listener
	s.listenAddr = listener.Addr().String()

	s.mu.Unlock()

	defer func() {
		s.mu.Lock()

		if s.listener == listener {
			s.listener = nil
			s.listenAddr = ""
		}

		s.mu.Unlock()

		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(
				err,
				net.ErrClosed,
			) {
				return nil
			}

			return fmt.Errorf(
				"%s: accept: %w",
				P2PServerVersionV1,
				err,
			)
		}

		// Existing P2P security boundary remains authoritative.
		if DefaultPeerSecurity != nil {
			ip := ExtractIP(
				conn.RemoteAddr(),
			)

			if err :=
				DefaultPeerSecurity.OnConnect(ip); err != nil {
				log.Printf(
					"p2p: rejecting inbound peer %s: %v",
					ip,
					err,
				)

				_ = conn.Close()
				continue
			}
		}

		registerPeer(conn)

		sendToPeer(
			conn,
			HandshakeMessage(),
		)

		go HandleConnection(
			conn,
			s.blockchain,
		)
	}
}

// Close stops accepting new inbound peers.
func (s *ServerV1) Close() error {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	listener := s.listener
	s.mu.RUnlock()

	if listener == nil {
		return nil
	}

	if err := listener.Close(); err != nil &&
		!errors.Is(err, net.ErrClosed) {
		return fmt.Errorf(
			"%s: close listener: %w",
			P2PServerVersionV1,
			err,
		)
	}

	return nil
}

// ListenAddress reports the actual bound listener address.
func (s *ServerV1) ListenAddress() string {
	if s == nil {
		return ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listenAddr
}

// IsRunning reports whether the inbound listener is active.
func (s *ServerV1) IsRunning() bool {
	if s == nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listener != nil
}
