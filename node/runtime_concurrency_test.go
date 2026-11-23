package node

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"path/filepath"
	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

func TestRuntimeReadOnlyDeniesMiningAndP2PAndSkipsPersistence(t *testing.T) {
	cfg, chain, _ := mandatoryStartupFixtureV1(t)
	if err := chain.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewRuntime(chain, cfg)
	runtime.SetAPIReadOnly(true)
	if err := runtime.SaveChain(); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.MineOnceV1("unused", 16); !errors.Is(err, ErrRuntimeReadOnlyV1) {
		t.Fatalf("read-only mine: %v", err)
	}
	if err := runtime.StartMining("unused", MiningHooksV1{Difficulty: func() int { return 16 }}); !errors.Is(err, ErrRuntimeReadOnlyV1) {
		t.Fatalf("read-only start: %v", err)
	}
	if err := runtime.ServeP2P("127.0.0.1:0"); !errors.Is(err, ErrRuntimeReadOnlyV1) {
		t.Fatalf("read-only P2P: %v", err)
	}
	if err := runtime.ConnectPeer("1", "127.0.0.1:1"); !errors.Is(err, ErrRuntimeReadOnlyV1) {
		t.Fatalf("read-only connect: %v", err)
	}
	after, err := os.ReadFile(cfg.ChainFile)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("observer wrote chain")
	}
}

func TestReloadCannotReplaceChainRetainedByP2P(t *testing.T) {
	cfg, chain, _ := mandatoryStartupFixtureV1(t)
	if err := chain.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	runtime := NewRuntime(chain, cfg)
	// Exercise the actual P2P lifecycle, rather than manufacturing its pin flag.
	serveDone := make(chan error, 1)
	go func() { serveDone <- runtime.ServeP2P("127.0.0.1:0") }()
	t.Cleanup(func() { _ = runtime.StopP2P() })
	deadline := time.Now().Add(10 * time.Second)
	for runtime.P2PListenAddress() == "" {
		select {
		case err := <-serveDone:
			t.Fatalf("P2P stopped before listening: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("P2P did not become ready")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := runtime.StopP2P(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("P2P did not stop")
	}
	// A peer handler may outlive its socket close; preserve its chain identity.
	runtime.SetAPIReadOnly(true)
	if changed, err := runtime.ReloadChainFromDiskIfChanged(); err == nil || changed {
		t.Fatal("P2P-pinned runtime accepted pointer replacement")
	}
	if current, _ := runtime.Snapshot(); current != chain {
		t.Fatal("pinned chain replaced")
	}
}

func TestMiningAutosaveAndShutdownPersistOneCoherentChain(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.BonusFile = filepath.Join(filepath.Dir(cfg.ChainFile), "bonus.json")
	t.Setenv("QC_NODE_DIR", t.TempDir())
	// Concurrency is the subject here, rather than PoW search variance. This
	// is the existing permitted local profile override, not a production bypass.
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	chain.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
	runtime := NewRuntime(chain, cfg)
	defer func() {
		runtime.StopMining()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := runtime.WaitMiningContext(ctx); err != nil {
			t.Error(err)
		}
	}()
	if err := runtime.SaveChain(); err != nil {
		t.Fatal(err)
	}
	failures := make(chan error, 100)
	autosaveDone := make(chan error, 1)
	autoContext, cancelAutosave := context.WithCancel(context.Background())
	defer cancelAutosave()
	go func() {
		autosaveDone <- runtime.RunAutosave(autoContext, 25*time.Millisecond, func(err error) { failures <- err })
	}()
	mined := make(chan struct{}, 1)
	miningErrors := make(chan error, 1)
	if bits, err := chain.ExpectedNextDifficultyV1(); err != nil || bits != 8 {
		t.Fatalf("local concurrency fixture difficulty=%d error=%v", bits, err)
	}
	if err := runtime.StartMining(wallet.NewWallet().GetAddress(), MiningHooksV1{OnError: func(err error) {
		select {
		case miningErrors <- err:
		default:
		}
	}, Difficulty: func() int { c, _ := runtime.Snapshot(); bits, _ := c.ExpectedNextDifficultyV1(); return bits }, OnBlockMined: func(*blockchain.Block) {
		select {
		case mined <- struct{}{}:
		default:
		}
	}}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-mined:
	case err := <-miningErrors:
		t.Fatalf("miner failed before persistence: %v", err)
	case <-time.After(20 * time.Second):
		runtime.StopMining()
		t.Fatal("miner did not produce a persisted block")
	}
	var saves sync.WaitGroup
	for i := 0; i < 3; i++ {
		saves.Add(1)
		go func() {
			defer saves.Done()
			for j := 0; j < 3; j++ {
				if err := runtime.SaveChain(); err != nil {
					failures <- err
				}
			}
		}()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	saves.Wait()
	select {
	case err := <-autosaveDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("autosave survived runtime shutdown")
	}
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if runtime.IsMining() {
		t.Fatal("miner survived shutdown")
	}
	if _, err := runtime.MineOnceV1("unused", 16); !errors.Is(err, ErrRuntimeShuttingDownV1) {
		t.Fatalf("shutdown admitted new work: %v", err)
	}
	loaded, err := blockchain.LoadBlockchainFromFileWithLocalContextV1(cfg.ChainFile, chain)
	if err != nil || loaded.BestHeightSnapshot() != chain.BestHeightSnapshot() {
		t.Fatalf("shutdown chain differs from runtime: %v", err)
	}
}
