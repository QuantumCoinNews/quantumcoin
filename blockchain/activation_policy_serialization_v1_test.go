package blockchain

import (
	"encoding/hex"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestBlockchainActivationPolicyV1IsNotSerializedV1(
	t *testing.T,
) {
	chain := NewBlockchain(1, 0)

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID: "quantumcoin-test-serialization-v1",

			LegacyGenesisHex: hex.EncodeToString(
				chain.Blocks[0].Hash,
			),

			ActivationHeight: 100,
			PreserveHistory:  true,
			PreserveBalances: true,
		},
	}

	if err := chain.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf(
			"bind source activation policy: %v",
			err,
		)
	}

	if got := chain.ProtocolAtHeightV1(100); got != qconsensus.ProtocolContinuityV2 {
		t.Fatalf(
			"source protocol = %q, want %q",
			got,
			qconsensus.ProtocolContinuityV2,
		)
	}

	data := SerializeBlockchain(chain)
	restored := DeserializeBlockchain(data)

	if got := restored.ProtocolAtHeightV1(100); got != qconsensus.ProtocolLegacyV1 {
		t.Fatalf(
			"serialized policy leaked into restored chain: %q",
			got,
		)
	}

	if err := restored.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf(
			"bind local policy after restore: %v",
			err,
		)
	}

	if got := restored.ProtocolAtHeightV1(100); got != qconsensus.ProtocolContinuityV2 {
		t.Fatalf(
			"restored locally bound protocol = %q, want %q",
			got,
			qconsensus.ProtocolContinuityV2,
		)
	}
}
