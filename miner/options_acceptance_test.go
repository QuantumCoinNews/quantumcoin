package miner

import (
	"sync/atomic"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestStartUsesEveryOptionPacketAndPreservesCallbacks(t *testing.T) {
	clearMinerRuntimeDifficultyEnvV1(t)
	t.Setenv("QC_NODE_DIR", t.TempDir())
	cfg := config.Default()
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	chain.TotalSupply = 1 // Exhausted cap: no block or reward-cache writes.
	reported := make(chan error, 1)
	var firstTick, lastTick, broadcasts atomic.Int32
	err := Start(chain, wallet.NewWallet().GetAddress(), cfg.DefaultDifficultyBits,
		Options{OnTick: func() { firstTick.Add(1) }, Animate: true},
		Options{OnError: func(err error) {
			select {
			case reported <- err:
			default:
			}
		}, Broadcaster: func(*blockchain.Block) { broadcasts.Add(1) }, Interval: 37 * time.Millisecond},
		Options{OnTick: func() { lastTick.Add(1) }, Interval: 41 * time.Millisecond, Animate: false},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Stop)
	minerLifecycleMu.Lock()
	got := state.opts
	minerLifecycleMu.Unlock()
	if got.Interval != 41*time.Millisecond || got.Animate || got.OnTick == nil || got.OnError == nil || got.Broadcaster == nil {
		t.Fatalf("option packets were lost: interval=%v animate=%v", got.Interval, got.Animate)
	}
	got.OnTick()
	got.Broadcaster(nil)
	if firstTick.Load() != 0 || lastTick.Load() != 1 || broadcasts.Load() != 1 {
		t.Fatal("wrong callback precedence")
	}
	select {
	case err := <-reported:
		if err == nil {
			t.Fatal("empty mining error")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("later OnError option did not reach running miner")
	}
	Stop()
	if IsActive() {
		t.Fatal("miner did not stop")
	}
	if chain.BestHeightSnapshot() != 0 {
		t.Fatal("test unexpectedly produced a block")
	}
}

func TestMergeOptionsV1DefaultsAndZeroValueSemantics(t *testing.T) {
	defaults := MergeOptionsV1()
	if defaults.Interval != 1200*time.Millisecond || !defaults.Animate {
		t.Fatal("startup defaults changed")
	}
	calls := 0
	opts := MergeOptionsV1(Options{Interval: 37 * time.Millisecond, Animate: true, OnTick: func() { calls++ }}, Options{Interval: -time.Second}, Options{})
	if opts.Interval != 37*time.Millisecond || opts.Animate || opts.OnTick == nil {
		t.Fatal("zero-value merge contract changed")
	}
	opts.OnTick()
	if calls != 1 {
		t.Fatal("empty packets cleared an earlier callback")
	}
}
