package p2p

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"

	"quantumcoin/blockchain"
)

// RunNode: TCP node başlatıcı
func RunNode(port string, bc *blockchain.Blockchain) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Panic(err)
	}
	defer listener.Close()

	fmt.Println("Node running on port:", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go HandleConnection(conn, bc)
	}
}

// ConnectToPeer: Diğer node’a bağlan
func ConnectToPeer(port string, address string, bc *blockchain.Blockchain) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Println("Connection failed:", err)
		return
	}
	fmt.Println("Connected to:", address)
	peers[address] = conn
	go HandleConnection(conn, bc)

	// Zinciri iste (sade haliyle)
	sendMessage(conn, Message{Type: MsgRequest, Data: nil})
}

func sendMessage(conn net.Conn, msg Message) {
	enc := gob.NewEncoder(conn)
	err := enc.Encode(msg)
	if err != nil {
		log.Println("Failed to send message:", err)
	}
}

// LoadPeersFromFile: Dosyadan peer adreslerini yükler
func LoadPeersFromFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	var list []string
	dec := gob.NewDecoder(file)
	err = dec.Decode(&list)
	if err != nil {
		return nil
	}
	return list
}
