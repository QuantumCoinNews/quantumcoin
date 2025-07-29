package miner

import (
	"log"
	"quantumcoin/blockchain"
)

func MineBlock(address string) *blockchain.Block {
	bc := blockchain.NewBlockchain(50, 25500000)
	block, err := bc.MineBlock(address, 16)
	if err != nil {
		log.Printf("Mining failed: %v", err)
		return nil
	}
	LogBlock(block)
	return block
}
