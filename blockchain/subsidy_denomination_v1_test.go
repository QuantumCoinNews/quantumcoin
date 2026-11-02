package blockchain

import (
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestLegacySubsidyUnitsRemainWholeQCV1(t *testing.T) {
	got, err := ExpectedSubsidyUnitsFromStateV1(
		qconsensus.ProtocolLegacyV1,
		1,
		0,
		int(QCTotalSupplyHardCap),
	)
	if err != nil {
		t.Fatalf("legacy subsidy units: %v", err)
	}

	if got != 50 {
		t.Fatalf(
			"legacy subsidy units=%d want=50",
			got,
		)
	}
}

func TestContinuityV2SubsidyUnitsRemainWholeQCV1(t *testing.T) {
	got, err := ExpectedSubsidyUnitsFromStateV1(
		qconsensus.ProtocolContinuityV2,
		1,
		0,
		int(QCTotalSupplyHardCap),
	)
	if err != nil {
		t.Fatalf("ContinuityV2 subsidy units: %v", err)
	}

	if got != 50 {
		t.Fatalf(
			"ContinuityV2 subsidy units=%d want=50",
			got,
		)
	}
}

func TestAtomV3SubsidyUsesAtomsV1(t *testing.T) {
	got, err := ExpectedSubsidyUnitsFromStateV1(
		qconsensus.ProtocolAtomV3,
		1,
		0,
		int(QCTotalSupplyHardCap),
	)
	if err != nil {
		t.Fatalf("AtomV3 subsidy units: %v", err)
	}

	if got != 50_000_000 {
		t.Fatalf(
			"AtomV3 subsidy units=%d want=50000000",
			got,
		)
	}
}

func TestSubsidyUnitBridgePreservesQCEmissionAccountingV1(
	t *testing.T,
) {
	const (
		totalSupplyQC = 25_500_000
		mintedQC      = 25_499_975
	)

	qcSubsidy := ExpectedSubsidyFromStateV1(
		1,
		mintedQC,
		totalSupplyQC,
	)

	if qcSubsidy != 25 {
		t.Fatalf(
			"QC subsidy=%d want=25",
			qcSubsidy,
		)
	}

	atomUnits, err :=
		ExpectedSubsidyUnitsFromStateV1(
			qconsensus.ProtocolAtomV3,
			1,
			mintedQC,
			totalSupplyQC,
		)
	if err != nil {
		t.Fatalf(
			"AtomV3 capped subsidy conversion: %v",
			err,
		)
	}

	if atomUnits != 25_000_000 {
		t.Fatalf(
			"AtomV3 capped subsidy=%d want=25000000",
			atomUnits,
		)
	}
}

func TestUnknownProtocolSubsidyUnitsFailClosedV1(
	t *testing.T,
) {
	_, err := ExpectedSubsidyUnitsFromStateV1(
		qconsensus.ProtocolVersion(
			"QC_UNKNOWN_PROTOCOL",
		),
		1,
		0,
		int(QCTotalSupplyHardCap),
	)

	if err == nil {
		t.Fatal(
			"unknown protocol subsidy conversion unexpectedly accepted",
		)
	}
}
