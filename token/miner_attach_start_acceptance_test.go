package token

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/miner"
	"quantumcoin/wallet"
)

func TestBridgeStartRejectsActiveMinerBeforeRegistryWrites(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "new registry"
		if existing {
			name = "existing registry"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("QC_ENABLE_TOKEN_MINER_BRIDGE", "1")
			t.Setenv("QC_DIFFICULTY_BITS", "")
			base := t.TempDir()
			t.Setenv("QC_NODE_DIR", base)
			dir := filepath.Join(base, "bridge")
			cfg := config.Default()
			chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
			chain.TotalSupply = 1 // Prevent actual block/cache writes.
			owner := canonicalAddressFixture("attach-owner")
			var before []byte
			if existing {
				registry := NewRegistry()
				if err := registry.Register(New("KEEP", "Existing Token", 8, 100, owner)); err != nil {
					t.Fatal(err)
				}
				if err := registry.Save(dir); err != nil {
					t.Fatal(err)
				}
				var err error
				before, err = os.ReadFile(filepath.Join(dir, registryFile))
				if err != nil {
					t.Fatal(err)
				}
			}
			originalErrors := make(chan error, 4)
			if err := miner.Start(chain, wallet.NewWallet().GetAddress(), cfg.DefaultDifficultyBits, miner.Options{Animate: false, OnError: func(err error) {
				select {
				case originalErrors <- err:
				default:
				}
			}}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(miner.Stop)
			var replacementErrors atomic.Int32
			bridgeCfg := MinerTokenConfig{Symbol: "NEW", Name: "New Token", Owner: owner, MaxSupply: 100, RewardPerBlock: 1, SaveDir: dir}
			bridge, err := StartMiningWithToken(chain, wallet.NewWallet().GetAddress(), cfg.DefaultDifficultyBits, bridgeCfg, miner.Options{Animate: false, OnError: func(error) { replacementErrors.Add(1) }})
			if bridge != nil || !errors.Is(err, ErrAttachToRunningMinerUnsupportedV1) || !errors.Is(err, miner.ErrMinerAlreadyActiveV1) {
				t.Fatalf("active start pretended to attach: bridge=%v error=%v", bridge != nil, err)
			}
			if !miner.IsActive() {
				t.Fatal("rejected attachment stopped original miner")
			}
			if existing {
				after, err := os.ReadFile(filepath.Join(dir, registryFile))
				if err != nil || !bytes.Equal(before, after) {
					t.Fatal("rejected attachment changed registry")
				}
			} else if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatal("rejected attachment created bridge directory or registry")
			}
			// Do not accept a notification queued before the rejected attempt
			// as proof that the original callback is still installed.
			for len(originalErrors) > 0 {
				<-originalErrors
			}
			select {
			case <-originalErrors:
			case <-time.After(3 * time.Second):
				t.Fatal("original error handler no longer works")
			}
			if replacementErrors.Load() != 0 {
				t.Fatal("rejected handler replaced original callbacks")
			}
			miner.Stop()
			started, err := StartMiningWithToken(chain, wallet.NewWallet().GetAddress(), cfg.DefaultDifficultyBits, bridgeCfg, miner.Options{Animate: false})
			if err != nil || started == nil || !miner.IsActive() {
				t.Fatalf("cold restart with bridge failed: %v", err)
			}
			miner.Stop()
			if started.tok.TotalSupply() != 0 || chain.BestHeightSnapshot() != 0 {
				t.Fatal("test unexpectedly produced a reward or block")
			}
		})
	}
}
