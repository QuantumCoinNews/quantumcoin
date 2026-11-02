package p2p

import (
	"fmt"
	"log"
	"net"
	"quantumcoin/blockchain"
	"strings"
	"time"
)

const OutboundDialTimeoutV1 = 5 * time.Second

// RunNode documents the QuantumCoin component used by this module.
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

// ConnectToPeer documents the QuantumCoin component used by this module.
// ConnectToPeer: Yalnızca uzak node'a outbound bağlantı kurar.
func ConnectToPeer(
	port string,
	address string,
	bc *blockchain.Blockchain,
) {
	_ = port

	if bc == nil {
		log.Println(
			"p2p: outbound connect rejected: blockchain unavailable",
		)
		return
	}

	address = strings.TrimSpace(address)
	if address == "" {
		log.Println(
			"p2p: outbound connect rejected: address empty",
		)
		return
	}

	conn, err := net.DialTimeout("tcp", address, OutboundDialTimeoutV1)
	if err != nil {
		log.Println(
			"Connection failed:",
			err,
		)
		return
	}

	fmt.Println(
		"Connected to:",
		address,
	)

	if DefaultPeerSecurity != nil {
		if ra := conn.RemoteAddr(); ra != nil {
			ip := ExtractIP(ra)

			if err :=
				DefaultPeerSecurity.OnConnect(ip); err != nil {
				log.Printf(
					"p2p: rejecting outbound peer %s: %v",
					ip,
					err,
				)

				_ = conn.Close()
				return
			}
		}
	}

	registerPeer(conn)

	sendToPeer(
		conn,
		HandshakeMessage(),
	)

	go HandleConnection(
		conn,
		bc,
	)

	sendToPeer(
		conn,
		RequestMessage(),
	)
}
