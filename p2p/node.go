package p2p

import (
	"fmt"
	"log"
	"net"
	"strings"

	"quantumcoin/blockchain"
)

// RunNode: TCP node başlatıcı
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
		registerPeer(conn)
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

	registerPeer(conn)
	go HandleConnection(conn, bc)

	// 3) Zinciri iste (sade haliyle)
	sendToPeer(conn, RequestMessage())
}
