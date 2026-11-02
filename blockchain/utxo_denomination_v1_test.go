package blockchain

import (
	"encoding/hex"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func atomDenominationTestChainV1(
	t *testing.T,
	activationHeight int,
) *Blockchain {
	t.Helper()

	bc := NewBlockchain(
		CanonicalInitialMiningRewardV1,
		int(QCTotalSupplyHardCap),
	)

	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	if activationHeight <= 0 {
		return bc
	}

	genesis := bc.Blocks[0]
	if genesis == nil || len(genesis.Hash) == 0 {
		t.Fatal("test chain missing genesis hash")
	}

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID:        "atom-denomination-test",
			LegacyGenesisHex: hex.EncodeToString(genesis.Hash),
			ActivationHeight: activationHeight,
			PreserveHistory:  true,
			PreserveBalances: true,
			TargetProtocol:   qconsensus.ProtocolAtomV3,
		},
	}

	if err := bc.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf("bind AtomV3 activation: %v", err)
	}

	return bc
}

func TestLegacyUTXOAmountRemainsWholeQCBeforeAtomActivationV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 100)

	got, err := bc.normalizeUTXOAmountForSpendV1(
		50,
		1,
		99,
	)
	if err != nil {
		t.Fatalf("normalize legacy UTXO: %v", err)
	}

	if got != 50 {
		t.Fatalf(
			"legacy amount=%d want=50",
			got,
		)
	}
}

func TestLegacyUTXOConvertsToAtomsAtAtomActivationV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 100)

	got, err := bc.normalizeUTXOAmountForSpendV1(
		50,
		99,
		100,
	)
	if err != nil {
		t.Fatalf("normalize legacy UTXO into AtomV3: %v", err)
	}

	if got != 50_000_000 {
		t.Fatalf(
			"legacy 50 QC normalized=%d want=50000000 atoms",
			got,
		)
	}
}

func TestAtomOriginAmountRemainsAtomsV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 100)

	got, err := bc.normalizeUTXOAmountForSpendV1(
		1_250_001,
		100,
		101,
	)
	if err != nil {
		t.Fatalf("normalize AtomV3 UTXO: %v", err)
	}

	if got != 1_250_001 {
		t.Fatalf(
			"AtomV3 amount=%d want=1250001",
			got,
		)
	}
}

func TestOneQCLegacyUTXOBecomesOneMillionAtomsV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 10)

	got, err := bc.normalizeUTXOAmountForSpendV1(
		1,
		9,
		10,
	)
	if err != nil {
		t.Fatalf("normalize 1 QC: %v", err)
	}

	if got != 1_000_000 {
		t.Fatalf(
			"1 QC normalized=%d want=1000000 atoms",
			got,
		)
	}
}

func TestUTXODenominationRejectsImpossibleOriginHeightV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 100)

	if _, err := bc.normalizeUTXOAmountForSpendV1(
		50,
		101,
		100,
	); err == nil {
		t.Fatal("future-origin UTXO unexpectedly accepted")
	}
}

func TestUTXODenominationRejectsNonPositiveAmountV1(
	t *testing.T,
) {
	bc := atomDenominationTestChainV1(t, 100)

	for _, raw := range []int{0, -1} {
		if _, err := bc.normalizeUTXOAmountForSpendV1(
			raw,
			1,
			100,
		); err == nil {
			t.Fatalf(
				"raw amount %d unexpectedly accepted",
				raw,
			)
		}
	}
}
