package miner

import (
	"sync"
	"time"
)

var (
	mu              sync.Mutex
	hashesTried     int
	startTime       time.Time
	lastBlockTime   time.Time
	lastBlockHashes int
)

func StartMetrics() {
	mu.Lock()
	defer mu.Unlock()
	hashesTried = 0
	startTime = time.Now()
}

func AddHashes(count int) {
	mu.Lock()
	defer mu.Unlock()
	hashesTried += count
}

func RecordBlock(hashCount int) {
	mu.Lock()
	defer mu.Unlock()
	lastBlockTime = time.Now()
	lastBlockHashes = hashCount
}

func GetHashRate() int {
	mu.Lock()
	defer mu.Unlock()
	duration := time.Since(startTime).Seconds()
	if duration == 0 {
		return 0
	}
	return int(float64(hashesTried) / duration)
}

func GetLastBlockMetrics() (string, int) {
	mu.Lock()
	defer mu.Unlock()
	return lastBlockTime.Format("02 Jan 2006 15:04:05"), lastBlockHashes
}
