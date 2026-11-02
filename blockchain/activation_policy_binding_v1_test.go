package blockchain

import (
	"encoding/hex"
	"strings"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestBlockchainActivationPolicyV1DefaultsToLegacyV1(
	t *testing.T,
) {
	chain := NewBlockchain(1, 0)

	if got := chain.ProtocolAtHeightV1(1_000_000); got != qconsensus.ProtocolLegacyV1 {
		t.Fatalf(
			"default protocol = %q, want %q",
			got,
			qconsensus.ProtocolLegacyV1,
		)
	}
}

func TestBlockchainActivationPolicyV1RejectsGenesisMismatchV1(
	t *testing.T,
) {
	chain := NewBlockchain(1, 0)

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID: "quantumcoin-test-binding-v1",

			LegacyGenesisHex: strings.Repeat("00", 32),

			ActivationHeight: 100,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}

	if err := chain.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err == nil {
		t.Fatal(
			"activation policy accepted mismatched genesis",
		)
	}

	if got := chain.ProtocolAtHeightV1(100); got != qconsensus.ProtocolLegacyV1 {
		t.Fatalf(
			"rejected policy changed protocol to %q",
			got,
		)
	}
}

func TestBlockchainActivationPolicyV1BindsMatchingPlanOnceV1(
	t *testing.T,
) {
	chain := NewBlockchain(1, 0)

	genesisHex := hex.EncodeToString(
		chain.Blocks[0].Hash,
	)

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "quantumcoin-test-binding-v1",
			LegacyGenesisHex: genesisHex,
			ActivationHeight: 100,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}

	if err := chain.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf(
			"bind matching activation policy: %v",
			err,
		)
	}

	if got := chain.ProtocolAtHeightV1(99); got != qconsensus.ProtocolLegacyV1 {
		t.Fatalf(
			"pre-activation protocol = %q",
			got,
		)
	}

	if got := chain.ProtocolAtHeightV1(100); got != qconsensus.ProtocolContinuityV2 {
		t.Fatalf(
			"activation protocol = %q",
			got,
		)
	}

	if err := chain.BindActivationPolicyV1(
		qconsensus.ActivationPolicyV1{},
	); err == nil {
		t.Fatal(
			"blockchain activation policy was rebound",
		)
	}
}
