package node

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

func mandatoryStartupFixtureV1(t *testing.T) (*config.Config, *blockchain.Blockchain, []byte) {
	t.Helper()
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.BonusFile = filepath.Join(filepath.Dir(cfg.ChainFile), "bonus.json")
	t.Setenv("QC_MAIN_ADDRESS", "")
	t.Setenv("QC_NODE_DIR", t.TempDir())
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	chain.SetCoinbaseMaturity(cfg.CoinbaseMaturity)
	difficulty, err := chain.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chain.MineBlock(wallet.NewWallet().GetAddress(), difficulty); err != nil {
		t.Fatal(err)
	}
	return cfg, chain, blockchain.SerializeBlockchain(chain)
}

func mutateStartupGobV1(t *testing.T, data []byte, mutate func(*blockchain.Blockchain)) []byte {
	t.Helper()
	var chain blockchain.Blockchain
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&chain); err != nil {
		t.Fatal(err)
	}
	mutate(&chain)
	return blockchain.SerializeBlockchain(&chain)
}

func invalidStartupFilesV1(t *testing.T, valid []byte) map[string][]byte {
	t.Helper()
	files := map[string][]byte{
		"invalid_gob":       []byte("not a chain"),
		"empty_file":        {},
		"truncated_gob":     append([]byte(nil), valid[:len(valid)/2]...),
		"trailing_garbage":  append(append([]byte(nil), valid...), []byte("unexpected trailing data")...),
		"second_gob_object": append(append([]byte(nil), valid...), valid...),
	}
	mutations := map[string]func(*blockchain.Blockchain){
		"empty_history":        func(c *blockchain.Blockchain) { c.Blocks = nil },
		"changed_genesis_hash": func(c *blockchain.Blockchain) { c.Blocks[0].Hash[0] ^= 1 },
		"changed_genesis_content": func(c *blockchain.Blockchain) {
			c.Blocks[0].Transactions[0].Outputs[0].Amount++
			c.Blocks[0].Transactions[0].ID = c.Blocks[0].Transactions[0].Hash()
		},
		"changed_supply":       func(c *blockchain.Blockchain) { c.TotalSupply++ },
		"broken_block_hash":    func(c *blockchain.Blockchain) { c.Blocks[1].Hash[0] ^= 1 },
		"broken_parent":        func(c *blockchain.Blockchain) { c.Blocks[1].PrevHash[0] ^= 1 },
		"wrong_index":          func(c *blockchain.Blockchain) { c.Blocks[1].Index++ },
		"wrong_difficulty":     func(c *blockchain.Blockchain) { c.Blocks[1].Difficulty++ },
		"missing_transactions": func(c *blockchain.Blockchain) { c.Blocks[1].Transactions = nil },
		"changed_coinbase":     func(c *blockchain.Blockchain) { c.Blocks[1].Transactions[0].Outputs[0].Amount++ },
	}
	for name, mutate := range mutations {
		files[name] = mutateStartupGobV1(t, valid, mutate)
	}
	return files
}

func TestDefaultBootstrapRejectsTamperedDiskChainWithoutStrictFlag(t *testing.T) {
	cfg, _, valid := mandatoryStartupFixtureV1(t)
	t.Setenv("QC_STRICT_CHAIN", "0")
	for name, data := range invalidStartupFilesV1(t, valid) {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(cfg.ChainFile, data, 0600); err != nil {
				t.Fatal(err)
			}
			runtime, err := BootstrapWithConfig(cfg, false)
			if err == nil || runtime != nil {
				t.Fatalf("unsafe disk chain published: runtime=%v err=%v", runtime, err)
			}
			after, readErr := os.ReadFile(cfg.ChainFile)
			if readErr != nil || !bytes.Equal(after, data) {
				t.Fatal("rejected chain was overwritten/reset")
			}
			entries, err := os.ReadDir(filepath.Dir(cfg.ChainFile))
			if err != nil || len(entries) != 1 {
				t.Fatalf("failed startup created a file: %v %v", entries, err)
			}
		})
	}
}

func TestBootstrapAlwaysValidatesAndRebuildsPersistedDerivedState(t *testing.T) {
	cfg, _, valid := mandatoryStartupFixtureV1(t)
	// Gob round trips normalize empty slices and time locations. Compare the
	// restored public history/state against a normalized replay, not Go-only
	// representation differences from the original in-memory miner object.
	source := blockchain.DeserializeBlockchain(valid)
	source.UpdateUTXOSet()
	poisoned := mutateStartupGobV1(t, valid, func(c *blockchain.Blockchain) {
		c.UTXO = map[string][]blockchain.TransactionOutput{"forged-balance": {{Amount: 999999, PubKeyHash: []byte("attacker")}}}
	})
	for _, env := range []string{"", "0", "1"} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("env=%q/option=%t", env, strict), func(t *testing.T) {
				t.Setenv("QC_STRICT_CHAIN", env)
				if err := os.WriteFile(cfg.ChainFile, poisoned, 0600); err != nil {
					t.Fatal(err)
				}
				runtime, err := BootstrapWithConfig(cfg, strict)
				if err != nil {
					t.Fatal(err)
				}
				chain, _ := runtime.Snapshot()
				if !reflect.DeepEqual(chain.AllBlocksSnapshot(), source.AllBlocksSnapshot()) || !reflect.DeepEqual(chain.UTXO, source.UTXO) {
					t.Fatal("startup changed validated history or trusted the file's forged UTXO")
				}
				if err := chain.ValidateBlockchainSynchronized(); err != nil {
					t.Fatal(err)
				}
				after, err := os.ReadFile(cfg.ChainFile)
				if err != nil || !bytes.Equal(after, poisoned) {
					t.Fatal("successful validation rewrote source file")
				}
			})
		}
	}
}

func TestReadonlyReloadPreservesHealthyRuntimeOnInvalidDiskChain(t *testing.T) {
	cfg, source, valid := mandatoryStartupFixtureV1(t)
	for name, data := range invalidStartupFilesV1(t, valid) {
		t.Run(name, func(t *testing.T) {
			runtime := NewRuntime(source, cfg)
			runtime.SetAPIReadOnly(true)
			if err := os.WriteFile(cfg.ChainFile, data, 0600); err != nil {
				t.Fatal(err)
			}
			changed, err := runtime.ReloadChainFromDiskIfChanged()
			if err == nil || changed {
				t.Fatalf("unsafe reload accepted: changed=%t err=%v", changed, err)
			}
			chain, _ := runtime.Snapshot()
			if chain != source || !runtime.ChainLastModifiedAt.IsZero() || runtime.ChainLastSize != 0 {
				t.Fatal("failed reload changed runtime or marked the file accepted")
			}
			after, err := os.ReadFile(cfg.ChainFile)
			if err != nil || !bytes.Equal(after, data) {
				t.Fatal("failed reload rewrote corrupt file")
			}
			if err := os.WriteFile(cfg.ChainFile, valid, 0600); err != nil {
				t.Fatal(err)
			}
			if changed, err := runtime.ReloadChainFromDiskIfChanged(); err != nil || !changed {
				t.Fatalf("valid repair could not be reloaded: %t %v", changed, err)
			}
		})
	}
}

func TestFreshBootstrapValidatesGenesisWithoutCreatingDiskState(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	cfg.BonusFile = filepath.Join(filepath.Dir(cfg.ChainFile), "bonus.json")
	t.Setenv("QC_STRICT_CHAIN", "0")
	runtime, err := BootstrapWithConfig(cfg, false)
	if err != nil || runtime == nil {
		t.Fatalf("fresh genesis bootstrap: %v", err)
	}
	if err := runtime.Blockchain.ValidateBlockchainSynchronized(); err != nil {
		t.Fatal(err)
	}
	if files, err := os.ReadDir(filepath.Dir(cfg.ChainFile)); err != nil || len(files) != 0 {
		t.Fatalf("fresh bootstrap wrote files: %v %v", files, err)
	}
}

func TestNFTStartupValidatesPersistedChainWithStrictOptionDisabled(t *testing.T) {
	f := nftStartupFixtureV1(t)
	t.Setenv("QC_STRICT_CHAIN", "0")
	runtime, err := bootstrapWithNFTStartupV1(f.cfg, false, &f.release, f.snapshotDir, &f.trust, f.manifest)
	if err != nil || runtime == nil || !runtime.NFTGameProgramSnapshotV1().Verified {
		t.Fatalf("default NFT startup failed: %v", err)
	}
	ownership, err := runtime.Blockchain.NFTOwnershipSnapshotSynchronizedV1()
	if err != nil || len(ownership.NFTs) != 1 {
		t.Fatalf("validated NFT ownership was lost: %+v %v", ownership, err)
	}
	f.producer.Blocks[1].Hash[0] ^= 1
	if err := f.producer.SaveToFile(f.cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(f.cfg.ChainFile)
	if err != nil {
		t.Fatal(err)
	}
	if runtime, err := bootstrapWithNFTStartupV1(f.cfg, false, &f.release, f.snapshotDir, &f.trust, f.manifest); err == nil || runtime != nil {
		t.Fatal("tampered NFT chain published with strict option disabled")
	}
	after, err := os.ReadFile(f.cfg.ChainFile)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("rejected NFT chain was overwritten")
	}
}
