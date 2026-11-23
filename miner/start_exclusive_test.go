package miner

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestExclusiveStartSerializesPreparationAndRejectsCompetingStart(t *testing.T) {
	clearMinerRuntimeDifficultyEnvV1(t)
	t.Setenv("QC_NODE_DIR", t.TempDir())
	cfg := config.Default()
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	chain.TotalSupply = 1
	address := wallet.NewWallet().GetAddress()
	entered, release := make(chan struct{}), make(chan struct{})
	first, second := make(chan error, 1), make(chan error, 1)
	var preparations atomic.Int32
	go func() {
		first <- StartExclusiveV1(chain, address, cfg.DefaultDifficultyBits, func() (Options, error) {
			preparations.Add(1)
			close(entered)
			<-release
			return Options{Animate: false}, nil
		})
	}()
	// Always unblock setup before cleanup attempts to acquire the lifecycle lock.
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		Stop()
	})
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("setup did not start")
	}
	go func() {
		second <- StartExclusiveV1(chain, address, cfg.DefaultDifficultyBits, func() (Options, error) { preparations.Add(1); return Options{Animate: false}, nil })
	}()
	close(release)
	select {
	case err := <-first:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("exclusive start did not finish")
	}
	select {
	case err := <-second:
		if !errors.Is(err, ErrMinerAlreadyActiveV1) {
			t.Fatalf("competing start result=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("competing start did not finish")
	}
	if preparations.Load() != 1 {
		t.Fatal("rejected start ran preparation")
	}
	Stop()
}

func TestExclusiveStartFailureDoesNotActivateMiner(t *testing.T) {
	clearMinerRuntimeDifficultyEnvV1(t)
	t.Setenv("QC_NODE_DIR", t.TempDir())
	cfg := config.Default()
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	chain.TotalSupply = 1
	address := wallet.NewWallet().GetAddress()
	sentinel := errors.New("preparation failed")
	if err := StartExclusiveV1(chain, address, cfg.DefaultDifficultyBits, func() (Options, error) { return Options{}, sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("setup error lost: %v", err)
	}
	if IsActive() {
		t.Fatal("failed setup activated miner")
	}
	if err := StartExclusiveV1(chain, address, cfg.DefaultDifficultyBits, nil); err == nil {
		t.Fatal("nil preparation accepted")
	}
	called := false
	if err := StartExclusiveV1(nil, address, cfg.DefaultDifficultyBits, func() (Options, error) { called = true; return Options{}, nil }); err == nil || called {
		t.Fatal("invalid request ran setup")
	}
}
