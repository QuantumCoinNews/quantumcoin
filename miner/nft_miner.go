package miner

import (
	"fmt"
	"time"
)

func GrantNFTReward(address string) {
	fmt.Printf("🎁 %s adresine nadir bir madencilik NFT'si verildi! (%s)\n", address, time.Now().Format("02 Jan 2006 15:04:05"))
}
