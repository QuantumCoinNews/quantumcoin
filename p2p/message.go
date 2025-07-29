package p2p

import (
	"bytes"
	"encoding/gob"
	"log"

	"quantumcoin/blockchain"
)

type MessageType string

const (
	MsgBlock   MessageType = "block"
	MsgTx      MessageType = "transaction"
	MsgChain   MessageType = "chain"
	MsgRequest MessageType = "request"
	// Ekstra: MsgPing, MsgPong, MsgPeerList, MsgError
)

// P2P mesaj yapısı
type Message struct {
	Type MessageType
	Data []byte
}

// SerializeMessage: Message -> []byte (gob)
func SerializeMessage(msg Message) []byte {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(msg)
	if err != nil {
		log.Panic("Mesaj kodlama hatası:", err)
	}
	return buffer.Bytes()
}

// DeserializeMessage: []byte -> Message (gob)
func DeserializeMessage(data []byte) Message {
	var msg Message
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&msg)
	if err != nil {
		log.Panic("Mesaj çözme hatası:", err)
	}
	return msg
}

// ChainMessage: Zincir paylaşımı
func ChainMessage(bc *blockchain.Blockchain) Message {
	data := blockchain.SerializeBlockchain(bc)
	return Message{
		Type: MsgChain,
		Data: data,
	}
}

// BlockMessage: Yeni blok paylaşımı
func BlockMessage(block *blockchain.Block) Message {
	return Message{
		Type: MsgBlock,
		Data: block.Serialize(),
	}
}

// TxMessage: İşlem paylaşımı
func TxMessage(tx *blockchain.Transaction) Message {
	return Message{
		Type: MsgTx,
		Data: tx.Serialize(),
	}
}

// RequestMessage: Zincir/veri isteği mesajı
func RequestMessage() Message {
	return Message{
		Type: MsgRequest,
		Data: nil,
	}
}
