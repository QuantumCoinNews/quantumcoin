package blockchain

import (
	"math"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestLegacyMiningRewardCompositionRemainsWholeQCV1(
	t *testing.T,
) {
	subsidy, total, err := ComposeMiningRewardUnitsV1(
		qconsensus.ProtocolLegacyV1,
		1,
		0,
		int(QCTotalSupplyHardCap),
		3,
	)
	if err != nil {
		t.Fatalf("legacy composition: %v", err)
	}

	if subsidy != 50 {
		t.Fatalf(
			"legacy subsidy=%d want=50",
			subsidy,
		)
	}

	if total != 53 {
		t.Fatalf(
			"legacy total=%d want=53",
			total,
		)
	}
}

func TestAtomV3MiningRewardCompositionUsesAtomUnitsV1(
	t *testing.T,
) {
	subsidy, total, err := ComposeMiningRewardUnitsV1(
		qconsensus.ProtocolAtomV3,
		1,
		0,
		int(QCTotalSupplyHardCap),
		1,
	)
	if err != nil {
		t.Fatalf("AtomV3 composition: %v", err)
	}

	if subsidy != 50_000_000 {
		t.Fatalf(
			"AtomV3 subsidy=%d want=50000000",
			subsidy,
		)
	}

	if total != 50_000_001 {
		t.Fatalf(
			"AtomV3 total=%d want=50000001",
			total,
		)
	}
}

func TestMiningRewardCompositionRejectsNegativeFeesV1(
	t *testing.T,
) {
	if _, _, err := ComposeMiningRewardUnitsV1(
		qconsensus.ProtocolAtomV3,
		1,
		0,
		int(QCTotalSupplyHardCap),
		-1,
	); err == nil {
		t.Fatal("negative verified fee unexpectedly accepted")
	}
}

func TestMiningRewardCompositionRejectsOverflowV1(
	t *testing.T,
) {
	if _, _, err := ComposeMiningRewardUnitsV1(
		qconsensus.ProtocolLegacyV1,
		1,
		0,
		int(QCTotalSupplyHardCap),
		math.MaxInt,
	); err == nil {
		t.Fatal("reward overflow unexpectedly accepted")
	}
}

func TestMiningRewardCompositionUnknownProtocolFailsClosedV1(
	t *testing.T,
) {
	if _, _, err := ComposeMiningRewardUnitsV1(
		qconsensus.ProtocolVersion("QC_UNKNOWN_PROTOCOL"),
		1,
		0,
		int(QCTotalSupplyHardCap),
		0,
	); err == nil {
		t.Fatal("unknown protocol unexpectedly accepted")
	}
}
