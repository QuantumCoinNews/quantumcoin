package p2p

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/security"
)

// peer: aynı bağlantı üzerinden eşzamanlı Encode yarışlarını önlemek için
type peer struct {
	conn net.Conn
	enc  *gob.Encoder
	mu   sync.Mutex // send lock
}

func (p *peer) send(msg Message) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("p2p: send on nil peer connection")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.conn.SetWriteDeadline(
		time.Now().Add(PeerWriteTimeoutV1),
	); err != nil {
		return fmt.Errorf("p2p: set write deadline: %w", err)
	}

	return p.enc.Encode(msg)
}

var (
	peersMu sync.Mutex
	peers   = make(map[string]*peer) // key: remote addr string
)

// --- Handshake durumu (addr bazlı) ---

var (
	handshakeMu sync.Mutex
	handshakeOK = make(map[string]bool) // key: conn.RemoteAddr().String()
)

func setHandshakeOK(addr string) {
	if addr == "" {
		return
	}
	handshakeMu.Lock()
	defer handshakeMu.Unlock()
	handshakeOK[addr] = true
}

func isHandshakeOK(addr string) bool {
	if addr == "" {
		return false
	}
	handshakeMu.Lock()
	defer handshakeMu.Unlock()
	return handshakeOK[addr]
}

// BroadcastMessage documents the QuantumCoin component used by this module.
// BroadcastMessage: Mesajı tüm bağlı peer’lara gönderir
func BroadcastMessage(msg Message) {
	peersMu.Lock()
	defer peersMu.Unlock()
	for addr, p := range peers {
		if err := p.send(msg); err != nil {
			log.Printf("Broadcast send to %s failed: %v", addr, err)
			logP2PReject("P2P_BROADCAST_SEND_FAILED", addr, err.Error(), map[string]any{})
			_ = p.conn.Close()
			delete(peers, addr)
		}
	}
}

// broadcastExcept: belirli bir kaynaktan gelmiş mesajı diğerlerine yay
func broadcastExcept(msg Message, except net.Addr) {
	peersMu.Lock()
	defer peersMu.Unlock()
	for addr, p := range peers {
		if except != nil && addr == except.String() {
			continue
		}
		if err := p.send(msg); err != nil {
			log.Printf("Broadcast(send) to %s failed: %v", addr, err)
			logP2PReject("P2P_BROADCAST_EXCEPT_SEND_FAILED", addr, err.Error(), map[string]any{})
			_ = p.conn.Close()
			delete(peers, addr)
		}
	}
}

// sendToPeer: Tek peer'a gönder
func sendToPeer(conn net.Conn, msg Message) {
	if conn == nil {
		logP2PReject("P2P_SEND_TO_NIL_CONN", "", "sendToPeer called with nil connection.", map[string]any{})
		return
	}

	peersMu.Lock()
	p := peers[conn.RemoteAddr().String()]
	peersMu.Unlock()

	if p == nil {
		// güvenlik: kayıtlı değilse geçici encoder ile deneyelim
		if err := gob.NewEncoder(conn).Encode(msg); err != nil {
			log.Println("Failed to send message:", err)
			logP2PReject("P2P_SEND_UNREGISTERED_PEER_FAILED", conn.RemoteAddr().String(), err.Error(), map[string]any{})
		}
		return
	}
	if err := p.send(msg); err != nil {
		log.Println("Failed to send message:", err)
		logP2PReject("P2P_SEND_REGISTERED_PEER_FAILED", conn.RemoteAddr().String(), err.Error(), map[string]any{})
	}
}

// HandleConnection documents the QuantumCoin component used by this module.
// HandleConnection: her yeni peer bağlantısı için reader loop
func HandleConnection(conn net.Conn, bc *blockchain.Blockchain) {
	if conn == nil {
		logP2PReject("P2P_NIL_CONNECTION", "", "HandleConnection called with nil connection.", map[string]any{})
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("p2p: panic from %s: %v", conn.RemoteAddr(), r)
			logP2PReject("P2P_CONNECTION_PANIC", conn.RemoteAddr().String(), fmt.Sprintf("%v", r), map[string]any{})
		}
		unregisterPeer(conn)
	}()

	// NOT: registerPeer(conn) artık node.go / ConnectToPeer içinde çağrılıyor.
	// Burada yalnızca okuma döngüsü var.

	handshakeTimer := startHandshakeTimeoutV1(
		conn,
		HandshakeTimeoutV1,
	)
	if handshakeTimer != nil {
		defer handshakeTimer.Stop()
	}

	heartbeatStop := make(chan struct{})
	go runPeerHeartbeatV1(
		conn,
		heartbeatStop,
		PeerHeartbeatIntervalV1,
	)
	defer close(heartbeatStop)

	dec := gob.NewDecoder(conn)
	for {
		if err := conn.SetReadDeadline(
			time.Now().Add(PeerReadIdleTimeoutV1),
		); err != nil {
			remote := conn.RemoteAddr().String()
			log.Printf(
				"p2p: set read deadline for %s failed: %v",
				remote,
				err,
			)
			logP2PReject(
				"P2P_READ_DEADLINE_FAILED",
				remote,
				err.Error(),
				map[string]any{},
			)
			return
		}

		var msg Message
		if err := dec.Decode(&msg); err != nil {
			if netErr, ok := err.(net.Error); ok &&
				netErr.Timeout() {
				remote := conn.RemoteAddr().String()
				log.Printf(
					"p2p: peer idle timeout from %s",
					remote,
				)
				logP2PReject(
					"P2P_PEER_IDLE_TIMEOUT",
					remote,
					"Peer sent no message before the idle timeout.",
					map[string]any{
						"timeoutMs": PeerReadIdleTimeoutV1.Milliseconds(),
					},
				)
				_ = conn.Close()
				return
			}
			log.Println("Connection closed or decode error:", err)
			logP2PReject("P2P_CONNECTION_DECODE_OR_CLOSE", conn.RemoteAddr().String(), err.Error(), map[string]any{})
			return
		}

		if err := ValidateP2PMessageSizeStrictV1(msg); err != nil {
			remote := conn.RemoteAddr().String()
			log.Printf("p2p: reject oversized message from %s: %v", remote, err)
			logP2PReject("P2P_MESSAGE_SIZE_REJECTED", remote, err.Error(), map[string]any{
				"type": string(msg.Type),
				"size": len(msg.Data),
			})
			_ = conn.Close()
			return
		}

		go handleMessage(msg, bc, conn)
	}
}

// PEER_P2P_REJECT_EVENTS_V1

// PEER_REPUTATION_INTEGRATION_V1
func peerReputationKey(subject string) string {
	if subject == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(subject); err == nil {
		return host
	}
	return subject
}

func adjustPeerReputationFromP2PEvent(eventType string, subject string) {
	if DefaultPeerSecurity == nil {
		return
	}

	peer := peerReputationKey(subject)
	if peer == "" {
		return
	}

	delta := 0

	switch eventType {
	case "P2P_INVALID_MAGIC_BANNED":
		delta = -30
	case "P2P_MESSAGE_SIZE_REJECTED":
		delta = -25
	case "P2P_MESSAGE_DROPPED_BY_SECURITY":
		delta = -25
	case "P2P_PEER_BLOCK_REJECTED":
		delta = -20
	case "P2P_CHAIN_INVALID_IGNORED":
		delta = -20
	case "P2P_VERSION_MISMATCH":
		delta = -15
	case "P2P_DUPLICATE_HANDSHAKE":
		delta = -10
	case "P2P_CHAIN_REPLACE_FAILED":
		delta = -15
	case "P2P_HANDSHAKE_TIMEOUT":
		delta = -15
	case "P2P_HANDSHAKE_DECODE_FAILED":
		delta = -10
	case "P2P_BLOCK_DECODE_FAILED":
		delta = -10
	case "P2P_TX_DECODE_FAILED":
		delta = -10
	case "P2P_TX_INVALID_SIGNATURE":
		delta = -10
	case "P2P_TX_REJECTED_BY_MEMPOOL":
		delta = -10
	case "P2P_CHAIN_DESERIALIZE_FAILED":
		delta = -10
	case "P2P_MESSAGE_BEFORE_HANDSHAKE":
		delta = -5
	case "P2P_UNKNOWN_MESSAGE_TYPE":
		delta = -5
	case "P2P_SEND_TO_NIL_CONN":
		delta = -3
	case "P2P_SEND_UNREGISTERED_PEER_FAILED":
		delta = -3
	case "P2P_SEND_REGISTERED_PEER_FAILED":
		delta = -3
	}

	if delta != 0 {
		DefaultPeerSecurity.AdjustReputation(peer, delta)
	}
}
func logP2PReject(eventType, subject, message string, meta map[string]any) {
	adjustPeerReputationFromP2PEvent(eventType, subject)
	_ = security.AppendEvent(".", security.NewEvent(
		eventType,
		security.SeverityWarning,
		"p2p",
		subject,
		message,
		meta,
	))
}

// handleMessage: Gelen mesaj türüne göre işlem
func handleMessage(msg Message, bc *blockchain.Blockchain, src net.Conn) {
	var remoteAddr string
	if src != nil && src.RemoteAddr() != nil {
		remoteAddr = src.RemoteAddr().String()
	}

	if err := ValidateP2PMessageSizeStrictV1(msg); err != nil {
		log.Printf("p2p: drop oversized message from %s: %v", remoteAddr, err)
		logP2PReject("P2P_MESSAGE_SIZE_REJECTED", remoteAddr, err.Error(), map[string]any{
			"type": string(msg.Type),
			"size": len(msg.Data),
		})
		if src != nil {
			_ = src.Close()
		}
		return
	}
	// --- IP bazlı flood + boyut limiti (security.go) ---
	if src != nil && src.RemoteAddr() != nil && DefaultPeerSecurity != nil {
		ip := ExtractIP(src.RemoteAddr())
		if err := DefaultPeerSecurity.AllowMessage(ip, len(msg.Data)); err != nil {
			log.Printf("p2p: drop message from %s (%s): %v", remoteAddr, ip, err)
			logP2PReject("P2P_MESSAGE_DROPPED_BY_SECURITY", ip, err.Error(), map[string]any{
				"remoteAddr": remoteAddr,
				"type":       string(msg.Type),
				"size":       len(msg.Data),
			})

			// Çok büyük mesaj veya flood: IP banlı ve bağlantıyı kapat
			if err == ErrMsgTooLarge || err == ErrRateLimited {
				_ = src.Close()
			}
			return
		}
	}

	// Handshake dışındaki tüm mesajlar için: önce handshake yapılmış mı?
	if msg.Type != MsgHello && !isHandshakeOK(remoteAddr) {
		log.Printf("p2p: drop message from %s before handshake (type=%s)", remoteAddr, msg.Type)
		logP2PReject("P2P_MESSAGE_BEFORE_HANDSHAKE", remoteAddr, "Message dropped before handshake.", map[string]any{
			"type": string(msg.Type),
		})
		return
	}

	switch msg.Type {
	case MsgHello:
		if isHandshakeOK(remoteAddr) {
			log.Printf(
				"p2p: duplicate handshake from %s",
				remoteAddr,
			)
			logP2PReject(
				"P2P_DUPLICATE_HANDSHAKE",
				remoteAddr,
				"Duplicate handshake on established peer session.",
				map[string]any{},
			)
			if src != nil {
				_ = src.Close()
			}
			return
		}

		// --- Handshake doğrulama ---
		var hs Handshake
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&hs); err != nil {
			log.Println("Handshake decode error:", err)
			logP2PReject("P2P_HANDSHAKE_DECODE_FAILED", remoteAddr, err.Error(), map[string]any{})
			return
		}

		expectedMagic := networkMagic()
		if hs.Magic == "" || hs.Magic != expectedMagic {
			ip := ""
			if src != nil && src.RemoteAddr() != nil {
				ip = ExtractIP(src.RemoteAddr())
			}
			log.Printf("p2p: invalid magic from %s: got=%q want=%q", ip, hs.Magic, expectedMagic)
			logP2PReject("P2P_INVALID_MAGIC_BANNED", ip, "Invalid network magic; peer banned.", map[string]any{
				"got":  hs.Magic,
				"want": expectedMagic,
			})
			if DefaultPeerSecurity != nil {
				DefaultPeerSecurity.Ban(ip, 0)
			}
			if src != nil {
				_ = src.Close()
			}
			return
		}

		if hs.Version != ProtocolVersion {
			log.Printf(
				"p2p: version mismatch from %s: got=%s want=%s",
				remoteAddr,
				hs.Version,
				ProtocolVersion,
			)

			logP2PReject(
				"P2P_VERSION_MISMATCH",
				remoteAddr,
				"Peer protocol version mismatch.",
				map[string]any{
					"got":  hs.Version,
					"want": ProtocolVersion,
				},
			)

			if src != nil {
				_ = src.Close()
			}

			return
		}

		setHandshakeOK(remoteAddr)
		log.Printf("p2p: handshake OK from %s (magic=%s, version=%s)", remoteAddr, hs.Magic, hs.Version)

	case MsgBlock:
		// --- Peer'dan gelen blok ---
		var blk blockchain.Block
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&blk); err != nil {
			log.Println("Block decode error:", err)
			logP2PReject("P2P_BLOCK_DECODE_FAILED", remoteAddr, err.Error(), map[string]any{})
			return
		}

		// Güvenli ekleme: index / prevHash / zaman / arz sınırı vs.
		if err := bc.AddBlockFromPeerSynchronizedV2(&blk); err != nil {
			log.Printf("Rejected peer block from %s: %v", remoteAddr, err)
			logP2PReject("P2P_PEER_BLOCK_REJECTED", remoteAddr, err.Error(), map[string]any{
				"height": blk.Index,
			})
			return
		}

		fmt.Printf("✓ New block accepted from peer %s (height=%d)\n", remoteAddr, blk.Index)

		// Bu bloğu diğer peer’lara yay ama göndereni hariç tut
		var except net.Addr
		if src != nil {
			except = src.RemoteAddr()
		}
		broadcastExcept(BlockMessage(&blk), except)

	case MsgTx:
		// --- Peer'dan gelen tx ---
		var tx blockchain.Transaction
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&tx); err != nil {
			log.Println("Transaction decode error:", err)
			logP2PReject("P2P_TX_DECODE_FAILED", remoteAddr, err.Error(), map[string]any{})
			return
		}
		// Çekirdek chain'e/tx havuzuna ekle
		if err := bc.AddTransactionFromPeerSynchronizedV1(&tx); err != nil {
			log.Printf("p2p: tx from %s rejected: %v", remoteAddr, err)
			logP2PReject("P2P_TX_REJECTED_BY_MEMPOOL", remoteAddr, err.Error(), map[string]any{})
			return
		}

		// Diğer peer’lara yay (kaynak hariç)
		var except net.Addr
		if src != nil {
			except = src.RemoteAddr()
		}
		broadcastExcept(TxMessage(&tx), except)

	case MsgChain:
		peerBC := blockchain.DeserializeBlockchain(msg.Data)
		if peerBC == nil {
			log.Println("peer chain deserialize failed")
			logP2PReject(
				"P2P_CHAIN_DESERIALIZE_FAILED",
				remoteAddr,
				"Peer chain deserialize failed.",
				map[string]any{},
			)
			return
		}

		if err := peerBC.BindPeerValidationContextFromLocalSynchronizedV1(bc); err != nil {
			log.Printf(
				"p2p: peer chain local validation context rejected from %s: %v",
				remoteAddr,
				err,
			)
			logP2PReject(
				"P2P_CHAIN_INVALID_IGNORED",
				remoteAddr,
				err.Error(),
				map[string]any{
					"stage": "local_validation_context_v1",
				},
			)
			return
		}

		if !peerBC.IsValidChainSynchronized() {
			log.Println("peer chain is invalid; ignoring")
			logP2PReject(
				"P2P_CHAIN_INVALID_IGNORED",
				remoteAddr,
				"Peer chain is invalid; ignored.",
				map[string]any{},
			)
			return
		}

		if peerBC.BestHeightSnapshot() <= bc.BestHeightSnapshot() {
			logP2PReject(
				"P2P_CHAIN_NOT_LONGER_IGNORED",
				remoteAddr,
				"Peer chain is not longer than local chain; ignored.",
				map[string]any{
					"peerHeight":  peerBC.BestHeightSnapshot(),
					"localHeight": bc.BestHeightSnapshot(),
				},
			)
			return
		}

		if err := bc.ReplaceChainFromPeerSynchronizedV1(
			peerBC.AllBlocksSnapshot(),
		); err != nil {
			log.Println("Chain replace failed:", err)
			logP2PReject(
				"P2P_CHAIN_REPLACE_FAILED",
				remoteAddr,
				err.Error(),
				map[string]any{},
			)
			return
		}

		fmt.Println("✓ Replaced chain with longer valid peer chain")

	case MsgRequest:
		// Sadece istekte bulunan peer'a yanıt
		sendToPeer(src, ChainMessage(bc))

	case MsgPing:
		sendToPeer(src, PongMessage())

	case MsgPong:
		// no-op

	default:
		log.Println("Unknown message type:", msg.Type)
		logP2PReject("P2P_UNKNOWN_MESSAGE_TYPE", remoteAddr, "Unknown peer message type.", map[string]any{
			"type": string(msg.Type),
		})
	}
}

// --- peer kayıt yönetimi ---
// DİKKAT: OnConnect/ban kontrolü artık node.go / ConnectToPeer içinde.
// Burada sadece peers map'i ve handshake state'i yönetiyoruz.

func registerPeer(conn net.Conn) {
	peersMu.Lock()
	defer peersMu.Unlock()
	peers[conn.RemoteAddr().String()] = &peer{
		conn: conn,
		enc:  gob.NewEncoder(conn),
	}
}

func unregisterPeer(conn net.Conn) {
	peersMu.Lock()
	defer peersMu.Unlock()

	addr := conn.RemoteAddr().String()
	if p, ok := peers[addr]; ok {
		_ = p.conn.Close()
		delete(peers, addr)
	}

	// Güvenlik tarafında connection sayısını düşür
	if DefaultPeerSecurity != nil {
		ip := ExtractIP(conn.RemoteAddr())
		DefaultPeerSecurity.OnDisconnect(ip)
	}

	// Handshake durumunu da temizle
	handshakeMu.Lock()
	delete(handshakeOK, addr)
	handshakeMu.Unlock()
}

func GetPeerCount() int {
	peersMu.Lock()
	defer peersMu.Unlock()
	return len(peers)
}

// CloseAllPeersV1 requests shutdown of all currently registered peer
// connections. Connections are copied while holding peersMu and closed
// after releasing the lock so HandleConnection cleanup can safely call
// unregisterPeer without deadlocking.
func CloseAllPeersV1() int {
	peersMu.Lock()

	conns := make([]net.Conn, 0, len(peers))

	for _, p := range peers {
		if p == nil || p.conn == nil {
			continue
		}

		conns = append(
			conns,
			p.conn,
		)
	}

	peersMu.Unlock()

	closed := 0

	for _, conn := range conns {
		if err := conn.Close(); err == nil {
			closed++
		}
	}

	return closed
}

const (
	HandshakeTimeoutV1      = 10 * time.Second
	PeerHeartbeatIntervalV1 = 30 * time.Second
	PeerReadIdleTimeoutV1   = 90 * time.Second
	PeerWriteTimeoutV1      = 15 * time.Second
)

func runPeerHeartbeatV1(
	conn net.Conn,
	stop <-chan struct{},
	interval time.Duration,
) {
	if conn == nil {
		return
	}
	if interval <= 0 {
		interval = PeerHeartbeatIntervalV1
	}

	remote := ""
	if conn.RemoteAddr() != nil {
		remote = conn.RemoteAddr().String()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if remote == "" || !isHandshakeOK(remote) {
				continue
			}

			peersMu.Lock()
			p := peers[remote]
			peersMu.Unlock()

			if p == nil {
				return
			}

			if err := p.send(PingMessage()); err != nil {
				log.Printf(
					"p2p: heartbeat send to %s failed: %v",
					remote,
					err,
				)
				logP2PReject(
					"P2P_HEARTBEAT_SEND_FAILED",
					remote,
					err.Error(),
					map[string]any{},
				)
				_ = conn.Close()
				return
			}
		}
	}
}

// startHandshakeTimeoutV1 closes peers that establish a TCP connection
// but fail to complete the QuantumCoin handshake within the allowed time.
func startHandshakeTimeoutV1(
	conn net.Conn,
	timeout time.Duration,
) *time.Timer {
	if conn == nil {
		return nil
	}

	if timeout <= 0 {
		timeout = HandshakeTimeoutV1
	}

	remoteAddr := ""
	if conn.RemoteAddr() != nil {
		remoteAddr = conn.RemoteAddr().String()
	}

	return time.AfterFunc(
		timeout,
		func() {
			if remoteAddr == "" ||
				isHandshakeOK(remoteAddr) {
				return
			}

			log.Printf(
				"p2p: handshake timeout from %s",
				remoteAddr,
			)

			logP2PReject(
				"P2P_HANDSHAKE_TIMEOUT",
				remoteAddr,
				"Peer did not complete handshake before timeout.",
				map[string]any{
					"timeoutMs": timeout.Milliseconds(),
				},
			)

			_ = conn.Close()
		},
	)
}
