package p2p

import (
	"fmt"
	"log"
	"net"
	"quantumcoin/blockchain"
	"strings"
)

// RunNode: TCP node başlatıcı (inbound bağlantılar için)
// DefaultPeerSecurity ile IP bazlı ban / connection limit kontrolü yapılır.
func RunNode(port string, bc *blockchain.Blockchain) {
	addr := port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("p2p listen %s failed: %v", addr, err)
	}
	defer listener.Close()

	fmt.Println("Node running on port:", strings.TrimPrefix(addr, ":"))

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}

		// --- GÜVENLİK: IP bazlı ban & connection limit ---
		if DefaultPeerSecurity != nil {
			ip := ExtractIP(conn.RemoteAddr())
			if err := DefaultPeerSecurity.OnConnect(ip); err != nil {
				log.Printf("p2p: rejecting inbound peer %s: %v", ip, err)
				_ = conn.Close()
				continue
			}
		}

		registerPeer(conn)

		// Bağlantı kurulur kurulmaz kendi handshake'imizi gönder
		sendToPeer(conn, HandshakeMessage())

		go HandleConnection(conn, bc)
	}
}

// ConnectToPeer: Diğer node’a bağlanır. Ek olarak yerel dinlemeyi de başlatır.
func ConnectToPeer(port string, address string, bc *blockchain.Blockchain) {
	// 1) Yerel dinlemeyi arka planda başlat
	go func() {
		defer func() {
			_ = recover() // zaten dinliyorsa panik olmasın
		}()
		RunNode(port, bc)
	}()

	// 2) Uzak düğüme bağlan
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Println("Connection failed:", err)
		return
	}
	fmt.Println("Connected to:", address)

	// --- GÜVENLİK: outbound peer için de kayıt ---
	if DefaultPeerSecurity != nil {
		if ra := conn.RemoteAddr(); ra != nil {
			ip := ExtractIP(ra)
			if err := DefaultPeerSecurity.OnConnect(ip); err != nil {
				log.Printf("p2p: rejecting outbound peer %s: %v", ip, err)
				_ = conn.Close()
				return
			}
		}
	}

	registerPeer(conn)

	// İlk iş olarak handshake gönder
	sendToPeer(conn, HandshakeMessage())

	go HandleConnection(conn, bc)

	// 3) Zinciri iste (sade haliyle)
	sendToPeer(conn, RequestMessage())
}
