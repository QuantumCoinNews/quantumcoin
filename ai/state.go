package ai

import "sync"

// tek yerden aç/kapa + blockchain hook'ları
var (
	mu      sync.RWMutex
	enabled = true
)

func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

func SetEnabled(v bool) {
	mu.Lock()
	enabled = v
	mu.Unlock()
}

// blockchain.NewBlockchain / AddBlock / ReplaceChain burayı çağırıyor.
// şimdilik no-op, ileride cache tutarsan burada temizlersin.
func OnBlockAdded(_ interface{})    {}
func OnChainReplaced(_ interface{}) {}
