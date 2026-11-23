package node

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quantumcoin/config"
)

func TestPolicyStartupRejectsDriftBeforeOpeningExistingChain(t *testing.T) {
	cases := map[string]func(*config.Config){
		"supply":         func(c *config.Config) { c.TotalSupply-- },
		"reward":         func(c *config.Config) { c.InitialReward++ },
		"vesting":        func(c *config.Config) { c.FounderVestingCliffMonths++ },
		"burn":           func(c *config.Config) { c.BurnAddress = "POLICY_DRIFT" },
		"mining_reserve": func(c *config.Config) { c.MiningPoolAddress = "POLICY_DRIFT" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := defaultRuntimePolicyConfigV1(t)
			cfg.BonusFile = filepath.Join(filepath.Dir(cfg.ChainFile), "audit", "bonus.jsonl")
			sentinel := []byte("existing chain sentinel: must not be decoded or replaced")
			if err := os.WriteFile(cfg.ChainFile, sentinel, 0600); err != nil {
				t.Fatal(err)
			}
			mutate(cfg)
			runtime, err := BootstrapWithConfig(cfg, false)
			if runtime != nil || err == nil || !strings.Contains(err.Error(), "runtime policy gate") {
				t.Fatalf("expected rejection at policy boundary, got runtime=%v error=%v", runtime, err)
			}
			after, err := os.ReadFile(cfg.ChainFile)
			if err != nil || !bytes.Equal(sentinel, after) {
				t.Fatal("policy rejection changed existing chain artifact")
			}
			for _, path := range []string{cfg.ChainFile + ".tmp", cfg.ChainFile + ".lock", filepath.Dir(cfg.BonusFile)} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("policy rejection touched runtime storage %s: %v", path, err)
				}
			}
		})
	}
}
