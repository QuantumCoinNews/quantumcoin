package blockchain

import (
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestBindCanonicalActivationPolicyV1BindsOnceV1(
	t *testing.T,
) {
	chain := NewBlockchain(1, 0)

	if err := chain.BindCanonicalActivationPolicyV1(); err != nil {
		t.Fatalf(
			"bind canonical activation policy: %v",
			err,
		)
	}

	if got := chain.ProtocolAtHeightV1(1_000_000); got != qconsensus.ProtocolLegacyV1 {
		t.Fatalf(
			"canonical protocol = %q, want %q",
			got,
			qconsensus.ProtocolLegacyV1,
		)
	}

	if err := chain.BindCanonicalActivationPolicyV1(); err == nil {
		t.Fatal(
			"canonical activation policy was bound twice",
		)
	}
}

func TestBindCanonicalActivationPolicyV1RejectsNilChainV1(
	t *testing.T,
) {
	var chain *Blockchain

	if err := chain.BindCanonicalActivationPolicyV1(); err == nil {
		t.Fatal(
			"nil blockchain accepted canonical activation policy",
		)
	}
}
