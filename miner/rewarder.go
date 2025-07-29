package miner

import (
	"fmt"
	"time"
)

type MinerActivity struct {
	Address     string
	BlocksMined int
	LastActive  time.Time
}

var minerStats = make(map[string]*MinerActivity)

func TrackMiner(address string) {
	if _, ok := minerStats[address]; !ok {
		minerStats[address] = &MinerActivity{Address: address}
	}
	minerStats[address].BlocksMined++
	minerStats[address].LastActive = time.Now()
}

func GetTopMiner() string {
	var top string
	var max int
	for addr, stat := range minerStats {
		if stat.BlocksMined > max {
			max = stat.BlocksMined
			top = addr
		}
	}
	return top
}

func PrintBonusMessage(address string) {
	fmt.Printf("🎉 %s adresine bonus ödül verildi! Toplam blok: %d\n", address, minerStats[address].BlocksMined)
}
