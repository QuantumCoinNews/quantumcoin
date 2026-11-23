package blockchain

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"quantumcoin/consensus"
	"quantumcoin/wallet"
)

type nonceCancelContextV1 struct {
	context.Context
	cancel context.CancelFunc
	checks atomic.Int32
}

func (c *nonceCancelContextV1) Err() error {
	if c.checks.Add(1) == 256 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestPoWContextCancelsDuringNonceSearch(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &nonceCancelContextV1{Context: parent, cancel: cancel}
	block := &Block{Index: 1, Timestamp: 1700000000, Miner: "test-miner", Difficulty: 255}
	nonce, hash, err := NewProofOfWork(block).RunContext(ctx)
	if !errors.Is(err, context.Canceled) || nonce != 0 || hash != nil || ctx.checks.Load() < 256 {
		t.Fatal("active hash search did not propagate cancellation", err)
	}
}

func TestPoWContextPreservesCanonicalNonceAndHash(t *testing.T) {
	block := &Block{Index: 1, Timestamp: 1700000000, Miner: "test-miner", Difficulty: 1}
	nonce, hash, err := NewProofOfWork(block).Run()
	if err != nil {
		t.Fatal(err)
	}
	got, witness, err := NewProofOfWork(block).RunContext(context.Background())
	if err != nil || got != nonce || !bytes.Equal(hash, witness) {
		t.Fatal("context changed canonical PoW")
	}
}

func TestMiningContextCancelsWhileWaitingForChainLock(t *testing.T) {
	chain := NewBlockchain(50, 0)
	chain.stateMu.Lock()
	defer chain.stateMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if block, err := chain.MineBlockContextV1(ctx, wallet.NewWallet().GetAddress(), 1); block != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("chain admission ignored cancellation", err)
	}
}

func TestCanceledPersistentMiningDoesNotInstallCandidate(t *testing.T) {
	chain := NewIsolatedLocalBlockchainV1(50, 0)
	file := filepath.Join(t.TempDir(), "chain.dat")
	if err := chain.SaveToFile(file); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	block, err := chain.MineBlockAndPersistContextV1(ctx, wallet.NewWallet().GetAddress(), 255, file)
	if block != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("canceled candidate returned success", err)
	}
	after, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(before, after) || chain.BestHeightSnapshot() != 0 {
		t.Fatal("cancellation changed disk or live chain")
	}
}

func TestCanceledProtocolBuildersStopNonceSearch(t *testing.T) {
	for _, protocol := range []consensus.ProtocolVersion{consensus.ProtocolLegacyV1, consensus.ProtocolContinuityV2, consensus.ProtocolAtomV3, consensus.ProtocolNFTV4} {
		t.Run(string(protocol), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()
			block, err := newBlockAtProtocolAndTimeContextV1(ctx, 1, nil, make([]byte, 32), wallet.NewWallet().GetAddress(), 255, nil, protocol, 1700000000)
			if block != nil || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("protocol search ignored cancellation", err)
			}
		})
	}
}
