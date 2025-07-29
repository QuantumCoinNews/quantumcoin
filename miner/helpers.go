package miner

import (
	"log"

	"quantumcoin/blockchain"
)

// MiningStatus geri dönen durum
type MiningStatus struct {
	HashesTried int64
	BlockHeight int
	Timestamp   int64
}

// LogBlock yeni blok çıktığında loglar
func LogBlock(b *blockchain.Block) {
	log.Printf("🚀 Yeni blok: Hash=%x  Index=%d", b.Hash, b.Index)
}
