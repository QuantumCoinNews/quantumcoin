package blockchain

import (
	"encoding/hex"
	"strings"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func canonicalAdapterDenominationChainV1(
	t *testing.T,
	target qconsensus.ProtocolVersion,
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

	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil {
		t.Fatal("missing genesis block")
	}

	if target == qconsensus.ProtocolLegacyV1 {
		return bc
	}

	policy := qconsensus.ActivationPolicyV1{
		Enabled: true,
		Plan: qconsensus.ActivationPlan{
			NetworkID: "canonical-adapter-denomination-test",

			LegacyGenesisHex: hex.EncodeToString(
				bc.Blocks[0].Hash,
			),

			ActivationHeight: activationHeight,
			PreserveHistory:  true,
			PreserveBalances: true,
			TargetProtocol:   target,
		},
	}

	if err := bc.BindActivationPolicyV1(verifiedActivationFixtureV1(t, policy)); err != nil {
		t.Fatalf(
			"bind activation policy: %v",
			err,
		)
	}

	return bc
}

func TestCanonicalAdapterSupplyUnitsLegacyV1(
	t *testing.T,
) {
	units, err :=
		qconsensus.WholeQCAmountToProtocolUnitsV1(
			qconsensus.ProtocolLegacyV1,
			QCTotalSupplyHardCap,
		)
	if err != nil {
		t.Fatalf(
			"legacy total supply conversion: %v",
			err,
		)
	}

	if units != 25_500_000 {
		t.Fatalf(
			"legacy supply units=%d want=25500000",
			units,
		)
	}
}

func TestCanonicalAdapterSupplyUnitsAtomV3V1(
	t *testing.T,
) {
	units, err :=
		qconsensus.WholeQCAmountToProtocolUnitsV1(
			qconsensus.ProtocolAtomV3,
			QCTotalSupplyHardCap,
		)
	if err != nil {
		t.Fatalf(
			"AtomV3 total supply conversion: %v",
			err,
		)
	}

	if units != 25_500_000_000_000 {
		t.Fatalf(
			"AtomV3 supply units=%d want=25500000000000",
			units,
		)
	}
}

func TestCanonicalAdapterSubsidyUnitsMatchProtocolV1(
	t *testing.T,
) {
	legacy, err := ExpectedSubsidyUnitsFromStateV1(
		qconsensus.ProtocolLegacyV1,
		1,
		0,
		int(QCTotalSupplyHardCap),
	)
	if err != nil {
		t.Fatalf("legacy subsidy units: %v", err)
	}

	atom, err := ExpectedSubsidyUnitsFromStateV1(
		qconsensus.ProtocolAtomV3,
		1,
		0,
		int(QCTotalSupplyHardCap),
	)
	if err != nil {
		t.Fatalf("AtomV3 subsidy units: %v", err)
	}

	if legacy != 50 {
		t.Fatalf(
			"legacy subsidy=%d want=50",
			legacy,
		)
	}

	if atom != 50_000_000 {
		t.Fatalf(
			"AtomV3 subsidy=%d want=50000000",
			atom,
		)
	}
}

func TestCanonicalAdapterAtomV3ReachesCanonicalComponentsV1(
	t *testing.T,
) {
	bc := canonicalAdapterDenominationChainV1(
		t,
		qconsensus.ProtocolAtomV3,
		1,
	)

	previous := bc.GetLastBlock()
	if previous == nil {
		t.Fatal("missing previous block")
	}

	// This intentionally incomplete block proves that AtomV3 now passes
	// protocol selection and reaches the canonical validation components.
	//
	// It must fail at PoW validation, not at protocol selection.
	block := &Block{
		Index:        1,
		Transactions: []*Transaction{},
		PrevHash:     append([]byte(nil), previous.Hash...),
		Difficulty:   1,
	}

	_, err := bc.ValidateCanonicalBlockEnvelopeV1(
		block,
		bc.TotalMinted(),
	)
	if err == nil {
		t.Fatal(
			"incomplete AtomV3 canonical block unexpectedly accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"proof-of-work validation failed",
	) {
		t.Fatalf(
			"AtomV3 did not reach canonical PoW validation: %v",
			err,
		)
	}

	if strings.Contains(
		err.Error(),
		"unsupported protocol",
	) {
		t.Fatalf(
			"AtomV3 was still rejected at protocol selection: %v",
			err,
		)
	}

	if strings.Contains(
		err.Error(),
		"protocol \"QC_ATOM_PROTOCOL_V3\" is not implemented",
	) {
		t.Fatalf(
			"AtomV3 was still rejected as unimplemented: %v",
			err,
		)
	}
}
