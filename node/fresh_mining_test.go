package node

import (
	"context"
	"errors"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestFreshCPUHashSearchReleasesRuntimeLocksAndCancels(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	t.Setenv("QC_NETWORK", "testnet-local")
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	chain, err := newRuntimeChainContextV1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic high-difficulty history is ONLY a cancellation/locking fixture.
	// It is never accepted, persisted, or presented as a valid PoW chain.
	start := time.Now().Add(-time.Hour * 2).Unix()
	for i := 1; i <= 110; i++ {
		chain.Blocks = append(chain.Blocks, &blockchain.Block{Index: i, Hash: make([]byte, 32), Timestamp: start + int64(i)*30, Difficulty: 255})
	}
	r := NewRuntime(chain, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := r.MineOnceContextV1(ctx, wallet.NewWallet().GetAddress(), 255); done <- err }()
	select {
	case err := <-done:
		t.Fatal("high-difficulty search ended unexpectedly", err)
	case <-time.After(50 * time.Millisecond):
	}
	query := make(chan error, 1)
	go func() {
		_ = r.MiningTip()
		_, _, err := r.PrepareExternalMining(wallet.NewWallet().GetAddress())
		query <- err
	}()
	select {
	case err := <-query:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CPU hashing blocked external work/tip")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CPU search ignored cancellation")
	}
}
