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

// HandleConnection: Gelen bağlantıyı dinler, mesajları işler
func HandleConnection(conn net.Conn, bc *blockchain.Blockchain) {
	defer func() {
		unregisterPeer(conn)
	}()

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
	switch msg.Type {
	case MsgBlock:
		var blk blockchain.Block
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&blk); err != nil {
			log.Println("Block decode error:", err)
			return
		}
		// Minimum doğrulama: PoW + bağlanırlık
		if err := bc.AddBlockFromPeer(&blk); err != nil {
			log.Printf("Rejected peer block: %v", err)
			return
		}
		fmt.Println("✓ New block accepted from peer")
		// Diğer peer’lara da (kaynak hariç) yay
		broadcastExcept(BlockMessage(&blk), src.RemoteAddr())

	case MsgTx:
		var tx blockchain.Transaction
		if err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&tx); err != nil {
			log.Println("Transaction decode error:", err)
			return
		}
		if !tx.Verify() {
			log.Println("Invalid tx from peer")
			return
		}
		// (Gelecekte: mempool'a ekle)
		// Şimdilik sadece tekrar yay
		broadcastExcept(TxMessage(&tx), src.RemoteAddr())

	case MsgChain:
		peerBC := blockchain.DeserializeBlockchain(msg.Data)
		// Basit kural: geçerli ve daha uzunsa değiştir
		if peerBC != nil && peerBC.IsValidChain() && peerBC.GetHeight() > bc.GetBestHeight() {
			if err := bc.ReplaceChain(peerBC.GetAllBlocks()); err != nil {
				log.Println("Chain replace failed:", err)
				return
			}
			fmt.Println("✓ Replaced chain with longer valid peer chain")
		}

	case MsgRequest:
		// Sadece istekte bulunan peer'a yanıtla
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
}
