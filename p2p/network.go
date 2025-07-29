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

var (
	peers = make(map[string]net.Conn)
	mu    sync.Mutex
)

// BroadcastMessage: Mesajı tüm bağlı peer’lara gönderir
func BroadcastMessage(msg Message) {
	mu.Lock()
	defer mu.Unlock()
	for addr, conn := range peers {
		enc := gob.NewEncoder(conn)
		err := enc.Encode(msg)
		if err != nil {
			log.Printf("Failed to send to %s: %v", addr, err)
			conn.Close()
			delete(peers, addr)
		}
	}
}

// HandleConnection: Gelen bağlantıyı dinler, mesajları işler
func HandleConnection(conn net.Conn, bc *blockchain.Blockchain) {
	dec := gob.NewDecoder(conn)
	for {
		var msg Message
		err := dec.Decode(&msg)
		if err != nil {
			log.Println("Connection closed or decode error:", err)
			conn.Close()
			return
		}
		go handleMessage(msg, bc)
	}
}

// handleMessage: Gelen mesaj türüne göre işlem
func handleMessage(msg Message, bc *blockchain.Blockchain) {
	switch msg.Type {
	case MsgBlock:
		var blk blockchain.Block
		err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&blk)
		if err != nil {
			log.Println("Block decode error:", err)
			return
		}
		// bc.AddBlockFromPeer(&blk) // Henüz yazılmadı
		fmt.Println("New block received from peer")

	case MsgTx:
		var tx blockchain.Transaction
		err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&tx)
		if err != nil {
			log.Println("Transaction decode error:", err)
			return
		}
		if tx.Verify() {
			fmt.Println("Valid transaction received from peer")
			// bc.AddTransaction(tx)
		}

	case MsgChain:
		var chain blockchain.Blockchain
		err := gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&chain)
		if err != nil {
			log.Println("Chain decode error:", err)
			return
		}
		// if chain.IsValidChain() && chain.GetHeight() > bc.GetHeight() {
		// 	bc.ReplaceChain(chain.Blocks)
		// 	fmt.Println("Replaced chain with longer peer chain")
		// }

	case MsgRequest:
		BroadcastMessage(ChainMessage(bc))

	default:
		log.Println("Unknown message type")
	}
}
