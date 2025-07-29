package miner

import (
	"time"

	"quantumcoin/blockchain"
)

var miningActive bool

func StartMining(minerAddress string, animationUpdate func(status MiningStatus)) {
	if miningActive {
		return
	}
	miningActive = true

	go func() {
		bc := blockchain.NewBlockchain(50, 25500000)
		for miningActive {
			start := time.Now()
			block, err := bc.MineBlock(minerAddress, 16)
			if err != nil {
				// Handle mining error, log if necessary
				continue
			}
			LogBlock(block)

			if animationUpdate != nil {
				animationUpdate(MiningStatus{
					HashesTried: 0,
					BlockHeight: block.Index,
					Timestamp:   start.Unix(),
				})
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

func StopMining() {
	miningActive = false
}

func IsMiningActive() bool {
	return miningActive
}
