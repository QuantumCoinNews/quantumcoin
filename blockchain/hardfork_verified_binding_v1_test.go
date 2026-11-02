package blockchain

import (
	"encoding/hex"
	"quantumcoin/consensus"
	"strings"
	"testing"
)

func TestHardforkBindingRejectsUnsignedActivationV1(t *testing.T) {
	chain := NewBlockchain(1, 0)
	policy := consensus.ActivationPolicyV1{Enabled: true, Plan: consensus.ActivationPlan{
		NetworkID: "test-only-hardfork-binding", LegacyGenesisHex: hex.EncodeToString(chain.Blocks[0].Hash),
		ActivationHeight: 100, TargetProtocol: consensus.ProtocolAtomV3,
		PreserveHistory: true, PreserveBalances: true,
	}}
	if err := chain.BindActivationPolicyV1(policy); err == nil {
		t.Fatal("unsigned manually-created plan enabled production blockchain activation")
	}
	if got := chain.ProtocolAtHeightV1(100); got != consensus.ProtocolLegacyV1 {
		t.Fatalf("rejected activation changed protocol: %s", got)
	}
}

func TestHardforkBindingRejectsModifiedVerifiedPlanV1(t *testing.T) {
	for _, field := range []string{"height", "protocol", "network", "genesis", "history", "balances", "proof"} {
		t.Run(field, func(t *testing.T) {
			chain := NewBlockchain(1, 0)
			p := verifiedActivationFixtureV1(t, consensus.ActivationPolicyV1{Enabled: true, Plan: consensus.ActivationPlan{
				NetworkID: "test-only-bound-plan", LegacyGenesisHex: hex.EncodeToString(chain.Blocks[0].Hash),
				ActivationHeight: 10, TargetProtocol: consensus.ProtocolAtomV3, PreserveHistory: true, PreserveBalances: true,
			}})
			bad := p
			switch field {
			case "height":
				bad.Plan.ActivationHeight++
			case "protocol":
				bad.Plan.TargetProtocol = consensus.ProtocolContinuityV2
			case "network":
				bad.Plan.NetworkID = "different"
			case "genesis":
				bad.Plan.LegacyGenesisHex = strings.Repeat("aa", 32)
			case "history":
				bad.Plan.PreserveHistory = false
			case "balances":
				bad.Plan.PreserveBalances = false
			case "proof":
				bad = consensus.ActivationPolicyV1{Enabled: true, Plan: p.Plan}
			}
			if err := chain.BindActivationPolicyV1(bad); err == nil {
				t.Fatal("modified plan bound")
			}
			if err := chain.BindActivationPolicyV1(p); err != nil {
				t.Fatalf("failed validation consumed binding: %v", err)
			}
			p.Plan.ActivationHeight = 1
			if chain.ProtocolAtHeightV1(9) != consensus.ProtocolLegacyV1 {
				t.Fatal("caller mutation changed bound plan")
			}
		})
	}
}

func TestHardforkBindingEnforcesSignedHeightInBlockValidationV1(t *testing.T) {
	chain := canonicalAdapterDenominationChainV1(t, consensus.ProtocolAtomV3, 2)
	for height := 1; height <= 3; height++ {
		expected := 50
		wrong := 50000000
		if height >= 2 {
			expected, wrong = wrong, expected
		}
		difficulty, err := chain.ExpectedNextDifficultyV1()
		if err != nil {
			t.Fatal(err)
		}
		previous := chain.GetLastBlock()
		invalid, err := NewBlock(height, []*Transaction{testPeerCoinbaseTxV2(wrong)}, previous.Hash, "fixture-miner", difficulty)
		if err != nil {
			t.Fatal(err)
		}
		if err := chain.AddBlockFromPeerStrictV2(invalid); err == nil {
			t.Fatalf("height %d accepted reward from wrong protocol", height)
		}
		valid, err := NewBlock(height, []*Transaction{testPeerCoinbaseTxV2(expected)}, previous.Hash, "fixture-miner", difficulty)
		if err != nil {
			t.Fatal(err)
		}
		if err := chain.AddBlockFromPeerStrictV2(valid); err != nil {
			t.Fatalf("height %d signed protocol rejected: %v", height, err)
		}
	}
}
