package node

import (
	"encoding/hex"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/consensus"
)

func TestHardforkRuntimeReloadPreservesVerifiedActivationV1(t *testing.T) {
	cfg := defaultRuntimePolicyConfigV1(t)
	chain := blockchain.NewBlockchain(cfg.InitialReward, cfg.TotalSupply)
	p := verifiedActivationFixtureV1(t, consensus.ActivationPolicyV1{Enabled: true, Plan: consensus.ActivationPlan{
		NetworkID: "test-only-reload", LegacyGenesisHex: hex.EncodeToString(chain.Blocks[0].Hash),
		ActivationHeight: 10, TargetProtocol: consensus.ProtocolAtomV3, PreserveHistory: true, PreserveBalances: true,
	}})
	if err := chain.BindActivationPolicyV1(p); err != nil {
		t.Fatal(err)
	}
	if err := chain.SaveToFile(cfg.ChainFile); err != nil {
		t.Fatal(err)
	}
	runtime := NewRuntime(chain, cfg)
	runtime.SetAPIReadOnly(true)
	reloaded, err := runtime.ReloadChainFromDiskIfChanged()
	if err != nil || !reloaded {
		t.Fatalf("reload: %v, %v", reloaded, err)
	}
	if got := runtime.Blockchain.ProtocolAtHeightV1(10); got != consensus.ProtocolAtomV3 {
		t.Fatalf("reload lost signed activation: got %s", got)
	}
}
