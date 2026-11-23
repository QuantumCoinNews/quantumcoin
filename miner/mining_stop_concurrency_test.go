package miner

import (
	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
	"sync"
	"testing"
	"time"
)

func TestCompatibilityMinerConcurrentStopAndRestart(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "")
	// A full emission cap makes MineBlock fail before any disk/advisory hooks.
	cfg := config.Default()
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	chain.TotalSupply = 1
	for round := 0; round < 4; round++ {
		if err := Start(chain, wallet.NewWallet().GetAddress(), cfg.DefaultDifficultyBits, Options{Animate: false}); err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			var wg sync.WaitGroup
			for i := 0; i < 32; i++ {
				wg.Add(1)
				go func() { defer wg.Done(); Stop() }()
			}
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("stop did not finish")
		}
		if IsActive() {
			t.Fatal("old miner retained after stop")
		}
	}
}
