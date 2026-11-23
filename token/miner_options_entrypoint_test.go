package token

import (
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/miner"
	"quantumcoin/wallet"
)

func TestMinerBridgeOptionsUseSharedStartupDefaults(t *testing.T) {
	bridge := new(MinerTokenBridge)
	got, want := bridge.Options(), miner.MergeOptionsV1()
	if got.Interval != want.Interval || got.Animate != want.Animate || got.OnBlock == nil {
		t.Fatal("bridge uses different startup defaults or lost its block adapter")
	}
}

func TestStartMiningWithTokenForwardsLaterErrorHandler(t *testing.T) {
	t.Setenv("QC_ENABLE_TOKEN_MINER_BRIDGE", "1")
	t.Setenv("QC_DIFFICULTY_BITS", "")
	dir := t.TempDir()
	t.Setenv("QC_NODE_DIR", dir)
	cfg := config.Default()
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	chain.TotalSupply = 1 // Real mining loop returns an error before block writes.
	reported := make(chan error, 1)
	bridge, err := StartMiningWithToken(chain, wallet.NewWallet().GetAddress(), cfg.DefaultDifficultyBits,
		MinerTokenConfig{Symbol: "OPT", Name: "Options Test", Owner: canonicalAddressFixture("options-owner"), MaxSupply: 100, RewardPerBlock: 1, SaveDir: dir},
		miner.Options{Animate: false},
		miner.Options{Animate: false, Interval: 37 * time.Millisecond, OnError: func(err error) {
			select {
			case reported <- err:
			default:
			}
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(miner.Stop)
	if bridge == nil {
		t.Fatal("missing token bridge")
	}
	select {
	case err := <-reported:
		if err == nil {
			t.Fatal("empty error callback")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("later option error handler did not reach miner loop")
	}
	miner.Stop()
	if chain.BestHeightSnapshot() != 0 || bridge.tok.TotalSupply() != 0 {
		t.Fatal("option test unexpectedly mined or issued a reward")
	}
}
