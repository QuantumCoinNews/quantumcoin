package blockchain

import (
	"strings"
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestReplacementMintedBridgePreservesLegacyWholeQCV1(
	t *testing.T,
) {
	got, err := ProtocolSubsidyUnitsToWholeQCV1(
		qconsensus.ProtocolLegacyV1,
		50,
	)
	if err != nil {
		t.Fatalf("legacy subsidy conversion: %v", err)
	}

	if got != 50 {
		t.Fatalf(
			"legacy subsidy QC=%d want=50",
			got,
		)
	}
}

func TestReplacementMintedBridgePreservesContinuityWholeQCV1(
	t *testing.T,
) {
	got, err := ProtocolSubsidyUnitsToWholeQCV1(
		qconsensus.ProtocolContinuityV2,
		25,
	)
	if err != nil {
		t.Fatalf(
			"ContinuityV2 subsidy conversion: %v",
			err,
		)
	}

	if got != 25 {
		t.Fatalf(
			"ContinuityV2 subsidy QC=%d want=25",
			got,
		)
	}
}

func TestReplacementMintedBridgeConvertsAtomV3ToWholeQCV1(
	t *testing.T,
) {
	got, err := ProtocolSubsidyUnitsToWholeQCV1(
		qconsensus.ProtocolAtomV3,
		50_000_000,
	)
	if err != nil {
		t.Fatalf("AtomV3 subsidy conversion: %v", err)
	}

	if got != 50 {
		t.Fatalf(
			"AtomV3 subsidy QC=%d want=50",
			got,
		)
	}
}

func TestReplacementMintedBridgeConvertsFinalPartialAtomRewardV1(
	t *testing.T,
) {
	got, err := ProtocolSubsidyUnitsToWholeQCV1(
		qconsensus.ProtocolAtomV3,
		7_000_000,
	)
	if err != nil {
		t.Fatalf(
			"AtomV3 partial subsidy conversion: %v",
			err,
		)
	}

	if got != 7 {
		t.Fatalf(
			"AtomV3 partial subsidy QC=%d want=7",
			got,
		)
	}
}

func TestReplacementMintedBridgeRejectsFractionalPolicyQCV1(
	t *testing.T,
) {
	_, err := ProtocolSubsidyUnitsToWholeQCV1(
		qconsensus.ProtocolAtomV3,
		50_000_001,
	)
	if err == nil {
		t.Fatal(
			"fractional whole-QC emission subsidy unexpectedly accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"exact whole-QC",
	) {
		t.Fatalf(
			"unexpected fractional subsidy error: %v",
			err,
		)
	}
}

func TestReplacementMintedBridgeRejectsNegativeV1(
	t *testing.T,
) {
	if _, err := ProtocolSubsidyUnitsToWholeQCV1(
		qconsensus.ProtocolAtomV3,
		-1,
	); err == nil {
		t.Fatal("negative subsidy unexpectedly accepted")
	}
}

func TestReplacementMintedBridgeRejectsUnknownProtocolV1(
	t *testing.T,
) {
	if _, err := ProtocolSubsidyUnitsToWholeQCV1(
		qconsensus.ProtocolVersion("UNKNOWN"),
		50,
	); err == nil {
		t.Fatal("unknown protocol unexpectedly accepted")
	}
}
