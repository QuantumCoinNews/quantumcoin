package token

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/security"
)

func TestMinerBridgeDeepReorgStopsNewHeadAndRestart(t *testing.T) {
	for _, restart := range []bool{false, true} {
		name := "new head"
		if restart {
			name = "restart"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			chain := minerBridgeTestChain(7, "miner", "old")
			bridge := minerBridgeTestBridge(t, chain, dir)
			old := chain.GetBlockByIndex(1)
			if err := bridge.processMinerTokenBlockV1(old, minerBridgeTestStatus(old)); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(filepath.Join(dir, registryFile))
			if err != nil {
				t.Fatal(err)
			}
			// A reorg preserves genesis and changes only the subsequent branch.
			replacement := []*blockchain.Block{chain.Blocks[0]}
			for height := 1; height <= 8; height++ {
				replacement = append(replacement, minerBridgeTestBlock(height, replacement[len(replacement)-1].Hash, canonicalAddressFixture("miner"), "new"))
			}
			chain.Blocks = replacement
			if restart {
				_, err = NewMinerTokenBridge(bridge.cfg, chain)
			} else {
				head := chain.GetBlockByIndex(8)
				err = bridge.processMinerTokenBlockV1(head, minerBridgeTestStatus(head))
			}
			if !errors.Is(err, ErrMinerTokenFinalizedHeightConflictV1) {
				t.Fatalf("deep reorg error=%v", err)
			}
			after, err := os.ReadFile(filepath.Join(dir, registryFile))
			if err != nil || string(before) != string(after) {
				t.Fatal("deep reorg changed ledger before stopping")
			}
			if bridge.tok.TotalSupply() != 25 {
				t.Fatal("deep reorg changed issued supply")
			}
		})
	}
}

func TestMinerBridgeLedgerLockRejectsAnotherProcess(t *testing.T) {
	if dir := os.Getenv("QC_MINER_TOKEN_LOCK_PROBE"); dir != "" {
		t.Setenv("QC_ENABLE_TOKEN_MINER_BRIDGE", "1")
		_, err := NewMinerTokenBridge(MinerTokenConfig{Symbol: "MINE", Name: "Mineable Test Token", Owner: canonicalAddressFixture("owner"), RewardPerBlock: 25, SaveDir: dir}, minerBridgeTestChain(7, "miner", "canonical"))
		if !errors.Is(err, ErrMinerTokenPersistenceV1) {
			t.Fatalf("another process bypassed ledger lock: %v", err)
		}
		return
	}
	dir := t.TempDir()
	chain := minerBridgeTestChain(7, "miner", "canonical")
	bridge := minerBridgeTestBridge(t, chain, dir)
	before, err := os.ReadFile(filepath.Join(dir, registryFile))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := lockMinerTokenLedgerV1(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if lock != nil {
			_ = lock.Release()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMinerBridgeLedgerLockRejectsAnotherProcess$")
	cmd.Env = append(os.Environ(), "QC_MINER_TOKEN_LOCK_PROBE="+dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("lock probe: %v; %s", err, output)
	}
	after, err := os.ReadFile(filepath.Join(dir, registryFile))
	if err != nil || string(before) != string(after) {
		t.Fatal("blocked child changed ledger")
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	lock = nil
	block := chain.GetBlockByIndex(1)
	if err := bridge.processMinerTokenBlockV1(block, minerBridgeTestStatus(block)); err != nil {
		t.Fatalf("released lock did not allow safe retry: %v", err)
	}
	if bridge.tok.TotalSupply() != 25 {
		t.Fatal("safe retry did not finalize once")
	}
}

func TestMinerBridgeIndependentInstancesCannotFinalizeSameRecordTwice(t *testing.T) {
	dir := t.TempDir()
	chain := minerBridgeTestChain(7, "miner", "canonical")
	first := minerBridgeTestBridge(t, chain, dir)
	second := minerBridgeTestBridge(t, chain, dir)
	block := chain.GetBlockByIndex(1)
	status := minerBridgeTestStatus(block)
	if err := first.processMinerTokenBlockV1(block, status); err != nil {
		t.Fatal(err)
	}
	if err := second.processMinerTokenBlockV1(block, status); err != nil && !errors.Is(err, ErrMinerTokenPersistenceV1) {
		t.Fatal(err)
	}
	finalized := 0
	for _, event := range security.ReadEvents(dir, 100).Events {
		if event.Type == TokenMinerRewardFinalizedEventV1 {
			finalized++
		}
	}
	if finalized != 1 {
		t.Fatalf("finalized events=%d, want 1", finalized)
	}
	restarted := minerBridgeTestBridge(t, chain, dir)
	if restarted.tok.TotalSupply() != 25 {
		t.Fatal("independent instances corrupted persistent supply")
	}
}
