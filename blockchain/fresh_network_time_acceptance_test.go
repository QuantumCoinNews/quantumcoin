package blockchain

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func freshTimeChain(t *testing.T, profile config.NetworkProfileV1) *Blockchain {
	t.Helper()
	cfg := config.Default()
	cfg.GenesisProfile = config.NetworkGenesisProfileV4
	chain, err := NewNetworkBlockchainV4(cfg, profile)
	if err != nil {
		t.Fatal(err)
	}
	return chain
}

func freshTimeBlock(t *testing.T, chain *Blockchain, miner string, timestamp int64, bits int) *Block {
	t.Helper()
	last := chain.BlockByIndexSnapshot(chain.BestHeightSnapshot())
	chain.stateMu.Lock()
	txs, err := chain.miningTransactionsLockedV1(miner, last.Index+1, time.Unix(timestamp, 0).UTC())
	chain.stateMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	block, err := newBlockAtTimeV1(last.Index+1, txs, last.Hash, miner, bits, timestamp)
	if err != nil {
		t.Fatal(err)
	}
	return block
}

func TestFreshV4MedianDifficultyBoundariesAndEnvironmentIndependence(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, profile := range []config.NetworkProfileV1{config.QCNetworkProfileTestnetLocalV1, config.QCNetworkProfileMainnetV1} {
		t.Run(string(profile), func(t *testing.T) {
			chain := freshTimeChain(t, profile)
			base := 16
			if profile == config.QCNetworkProfileTestnetLocalV1 {
				base = 8
			}
			for _, row := range []struct {
				name          string
				seconds       int64
				current, want int
			}{
				{"fast", 20, base, base + 1}, {"lower_boundary", 24, base + 1, base + 1},
				{"target", 30, base + 1, base + 1}, {"upper_boundary", 36, base + 1, base + 1},
				{"slow", 40, base + 1, base}, {"floor", 40, base, base}, {"ceiling", 20, 255, 255},
			} {
				t.Run(row.name, func(t *testing.T) {
					// Detached histories exercise time-window boundaries without
					// pretending synthetic headers are validated PoW chains.
					chain.Blocks = nil
					for i := 0; i < 111; i++ {
						chain.Blocks = append(chain.Blocks, &Block{Index: i, Timestamp: 1_700_000_000 + int64(i)*row.seconds, Difficulty: row.current})
					}
					chain.Blocks[110].Timestamp += 7200
					for _, override := range []string{"", "1", "30", "invalid"} {
						t.Setenv("QC_DIFFICULTY_BITS", override)
						got, err := chain.ExpectedNextDifficultyV1()
						if err != nil || got != row.want {
							t.Fatalf("bits=%d want=%d override=%q: %v", got, row.want, override, err)
						}
					}
					chain.Blocks = chain.Blocks[:110]
					if got, _ := chain.ExpectedNextDifficultyV1(); got != base {
						t.Fatal("retarget activated before both full median windows")
					}
				})
			}
		})
	}
}

func TestFreshV4ValidPoWStillRejectsWrongTimeOrDifficultyEverywhere(t *testing.T) {
	t.Chdir(t.TempDir())
	chain := freshTimeChain(t, config.QCNetworkProfileTestnetLocalV1)
	miner := wallet.NewWallet().GetAddress()
	start := chain.BlockByIndexSnapshot(0).Timestamp
	for i := 1; i <= 12; i++ {
		block := freshTimeBlock(t, chain, miner, start+int64(i)*30, 8)
		if err := chain.AddBlockFromPeerStrictV2(block); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []struct {
		name      string
		timestamp int64
		bits      int
		reason    string
	}{
		{"equal_median", medianBlockTimeV2(chain.Blocks, 12), 8, "median time past"},
		{"older_than_median", medianBlockTimeV2(chain.Blocks, 12) - 1, 8, "median time past"},
		{"easier_pow", start + 390, 7, "difficulty"},
		{"harder_pow", start + 390, 9, "difficulty"},
		{"future_clock", time.Now().Add(3 * time.Hour).Unix(), 8, "future"},
	} {
		t.Run(row.name, func(t *testing.T) {
			// A distinct payee keeps a bad-time fixture from accidentally
			// duplicating an earlier coinbase before reaching the time check.
			bad := freshTimeBlock(t, chain, wallet.NewWallet().GetAddress(), row.timestamp, row.bits)
			if !NewProofOfWork(bad).Validate() {
				t.Fatal("negative fixture has invalid PoW")
			}
			candidate := append(chain.BlocksSnapshot(), bad)
			for _, err := range []error{chain.ValidatePeerBlockStrictV2(bad), chain.ValidateReplacementChainStrictV1(candidate)} {
				if err == nil || !strings.Contains(err.Error(), row.reason) {
					t.Fatalf("expected %s rejection: %v", row.reason, err)
				}
			}
			forged := freshTimeChain(t, config.QCNetworkProfileTestnetLocalV1)
			forged.Blocks = candidate // deliberately invalid disk fixture, never runtime state
			path := filepath.Join(t.TempDir(), "invalid-chain.dat")
			if err := forged.SaveToFile(path); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = LoadBlockchainFromFileWithLocalContextV1(path, chain); err == nil || !strings.Contains(err.Error(), row.reason) {
				t.Fatalf("disk accepted invalid %s: %v", row.name, err)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("rejected disk fixture was modified", err)
			}
			if chain.BestHeightSnapshot() != 12 {
				t.Fatal("negative validation mutated live test chain")
			}
		})
	}
}

func TestFreshV4NodesAndRestartAgreeAcrossMedianRetarget(t *testing.T) {
	t.Chdir(t.TempDir())
	chain := freshTimeChain(t, config.QCNetworkProfileTestnetLocalV1)
	peer := freshTimeChain(t, config.QCNetworkProfileTestnetPublicV1)
	miner := wallet.NewWallet().GetAddress()
	start := chain.BlockByIndexSnapshot(0).Timestamp
	path := filepath.Join(t.TempDir(), "valid-chain.dat")
	for height := 1; height <= 112; height++ {
		bits, err := chain.ExpectedNextDifficultyV1()
		if err != nil {
			t.Fatal(err)
		}
		if other, _ := peer.ExpectedNextDifficultyV1(); other != bits {
			t.Fatal("nodes disagree on next difficulty")
		}
		if height == 111 && bits != 9 {
			t.Fatalf("first full median-window retarget: got %d want 9", bits)
		}
		block := freshTimeBlock(t, chain, miner, start+int64(height)*20, bits)
		if err = chain.AddBlockFromPeerStrictV2(block); err != nil {
			t.Fatal(err)
		}
		if err = peer.AddBlockFromPeerStrictV2(cloneBlockV1(block)); err != nil {
			t.Fatal(err)
		}
		if height == 110 || height == 112 {
			if err = chain.SaveToFile(path); err != nil {
				t.Fatal(err)
			}
			restarted, err := LoadBlockchainFromFileWithLocalContextV1(path, freshTimeChain(t, config.QCNetworkProfileTestnetLocalV1))
			if err != nil {
				t.Fatal(err)
			}
			chain = restarted
		}
	}
	if !bytes.Equal(chain.BlockByIndexSnapshot(112).Hash, peer.BlockByIndexSnapshot(112).Hash) {
		t.Fatal("restart changed selected tip")
	}
	bits, _ := chain.ExpectedNextDifficultyV1()
	template, _, err := chain.PrepareExternalMiningTemplate(miner, bits)
	if err != nil || template.Timestamp <= medianBlockTimeV2(chain.Blocks, 112) || template.Difficulty != bits {
		t.Fatal("external mining disagrees with canonical time/difficulty", err)
	}
	if _, _, err = chain.PrepareExternalMiningTemplate(miner, bits-1); err == nil {
		t.Fatal("external mining accepted easier requested difficulty")
	}
	if _, err = chain.MineBlock(miner, bits-1); err == nil {
		t.Fatal("local mining accepted easier requested difficulty")
	}
	if _, err = chain.MineBlock(miner, bits); err != nil {
		t.Fatal("valid local mining failed", err)
	}
}
