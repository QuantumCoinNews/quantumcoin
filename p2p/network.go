package p2p

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"

	"quantumcoin/blockchain"
)

// peer: aynı bağlantı üzerinden eşzamanlı Encode yarışlarını önlemek için
type peer struct {
	conn net.Conn
	enc  *gob.Encoder
	mu   sync.Mutex // send lock
}

func (p *peer) send(msg Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()
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

// BroadcastMessage: Mesajı tüm bağlı peer’lara gönderir
func BroadcastMessage(msg Message) {
	peersMu.Lock()
	defer peersMu.Unlock()
	for addr, p := range peers {
		if err := p.send(msg); err != nil {
			log.Printf("Broadcast send to %s failed: %v", addr, err)
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
			_ = p.conn.Close()
			delete(peers, addr)
		}
	}
}

// sendToPeer: Tek peer'a gönder
func sendToPeer(conn net.Conn, msg Message) {
	peersMu.Lock()
	p := peers[conn.RemoteAddr().String()]
	peersMu.Unlock()

	if p == nil {
		// güvenlik: kayıtlı değilse geçici encoder ile deneyelim
		if err := gob.NewEncoder(conn).Encode(msg); err != nil {
			log.Println("Failed to send message:", err)
		}
		return
	}
	if err := p.send(msg); err != nil {
		log.Println("Failed to send message:", err)
	}
}

// HandleConnection: her yeni peer bağlantısı için reader loop
func HandleConnection(conn net.Conn, bc *blockchain.Blockchain) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("p2p: panic from %s: %v", conn.RemoteAddr(), r)
		}
		unregisterPeer(conn)
	}()

	// NOT: registerPeer(conn) artık node.go / ConnectToPeer içinde çağrılıyor.
	// Burada yalnızca okuma döngüsü var.

	dec := gob.NewDecoder(conn)
	for {
		var msg Message
		if err := dec.Decode(&msg); err != nil {
			log.Println("Connection closed or decode error:", err)
			return
		}
		go handleMessage(msg, bc, conn)
	}
}

// handleMessage: Gelen mesaj türüne göre işlem
func handleMessage(msg Message, bc *blockchain.Blockchain, src net.Conn) {
	var remoteAddr string
	if src != nil && src.RemoteAddr() != nil {
		remoteAddr = src.RemoteAddr().String()
	}

	// --- IP bazlı flood + boyut limiti (security.go) ---
	if src != nil && src.RemoteAddr() != nil && DefaultPeerSecurity != nil {
		ip := ExtractIP(src.RemoteAddr())
		if err := DefaultPeerSecurity.AllowMessage(ip, len(msg.Data)); err != nil {
			log.Printf("p2p: drop message from %s (%s): %v", remoteAddr, ip, err)

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
		return
	}

	switch msg.Type {
	case MsgHello:
		// --- Handshake doğrulama ---
		var hs Handshake
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&hs); err != nil {
			log.Println("Handshake decode error:", err)
			return
		}

		expectedMagic := networkMagic()
		if hs.Magic == "" || hs.Magic != expectedMagic {
			ip := ExtractIP(src.RemoteAddr())
			log.Printf("p2p: invalid magic from %s: got=%q want=%q", ip, hs.Magic, expectedMagic)
			if DefaultPeerSecurity != nil {
				DefaultPeerSecurity.Ban(ip, 0)
			}
			_ = src.Close()
			return
		}

		if hs.Version != ProtocolVersion {
			log.Printf("p2p: version mismatch from %s: got=%s want=%s",
				remoteAddr, hs.Version, ProtocolVersion)
			// Şimdilik sadece log; ileride daha katı yapabiliriz.
		}

		setHandshakeOK(remoteAddr)
		log.Printf("p2p: handshake OK from %s (magic=%s, version=%s)", remoteAddr, hs.Magic, hs.Version)

	case MsgBlock:
		// --- Peer'dan gelen blok ---
		var blk blockchain.Block
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&blk); err != nil {
			log.Println("Block decode error:", err)
			return
		}

		// Güvenli ekleme: index / prevHash / zaman / arz sınırı vs.
		if err := bc.AddBlockFromPeerSecure(&blk); err != nil {
			log.Printf("Rejected peer block from %s: %v", remoteAddr, err)
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
			return
		}

		if !tx.Verify() {
			log.Printf("Invalid tx from %s: signature verify failed", remoteAddr)
			return
		}

		// Çekirdek chain'e/tx havuzuna ekle
		if err := bc.AddTransaction(&tx); err != nil {
			log.Printf("p2p: tx from %s rejected: %v", remoteAddr, err)
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
			return
		}

		if !peerBC.IsValidChain() {
			log.Println("peer chain is invalid; ignoring")
			return
		}

		if peerBC.GetHeight() <= bc.GetBestHeight() {
			return
		}

		if err := bc.ReplaceChain(peerBC.GetAllBlocks()); err != nil {
			log.Println("Chain replace failed:", err)
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
