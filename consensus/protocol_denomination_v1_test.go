package consensus

import (
	"strconv"
	"testing"
)

func TestLegacyProtocolUsesWholeQCIntegerV1(t *testing.T) {
	unit, err := MonetaryUnitAtProtocolV1(
		ProtocolLegacyV1,
	)
	if err != nil {
		t.Fatalf("legacy monetary unit: %v", err)
	}

	if unit != MonetaryUnitWholeQCV1 {
		t.Fatalf(
			"legacy monetary unit=%q want=%q",
			unit,
			MonetaryUnitWholeQCV1,
		)
	}

	if ProtocolUsesAtomDenominationV1(
		ProtocolLegacyV1,
	) {
		t.Fatal("LegacyV1 unexpectedly uses atom denomination")
	}

	got, err := WholeQCAmountToProtocolUnitsV1(
		ProtocolLegacyV1,
		50,
	)
	if err != nil {
		t.Fatalf("legacy reward conversion: %v", err)
	}

	if got != 50 {
		t.Fatalf(
			"legacy 50 QC units=%d want=50",
			got,
		)
	}
}

func TestContinuityV2PreservesWholeQCDenominationV1(t *testing.T) {
	unit, err := MonetaryUnitAtProtocolV1(
		ProtocolContinuityV2,
	)
	if err != nil {
		t.Fatalf("continuity monetary unit: %v", err)
	}

	if unit != MonetaryUnitWholeQCV1 {
		t.Fatalf(
			"ContinuityV2 monetary unit=%q want=%q",
			unit,
			MonetaryUnitWholeQCV1,
		)
	}
}

func TestAtomV3UsesCanonicalAtomDenominationV1(t *testing.T) {
	if strconv.IntSize != 64 {
		t.Skip("AtomV3 requires 64-bit int runtime")
	}

	unit, err := MonetaryUnitAtProtocolV1(
		ProtocolAtomV3,
	)
	if err != nil {
		t.Fatalf("AtomV3 monetary unit: %v", err)
	}

	if unit != MonetaryUnitAtomV1 {
		t.Fatalf(
			"AtomV3 monetary unit=%q want=%q",
			unit,
			MonetaryUnitAtomV1,
		)
	}

	if !ProtocolUsesAtomDenominationV1(
		ProtocolAtomV3,
	) {
		t.Fatal("AtomV3 did not select atom denomination")
	}

	reward, err := WholeQCAmountToProtocolUnitsV1(
		ProtocolAtomV3,
		50,
	)
	if err != nil {
		t.Fatalf("AtomV3 reward conversion: %v", err)
	}

	if reward != 50_000_000 {
		t.Fatalf(
			"AtomV3 50 QC units=%d want=50000000",
			reward,
		)
	}

	supply, err := WholeQCAmountToProtocolUnitsV1(
		ProtocolAtomV3,
		25_500_000,
	)
	if err != nil {
		t.Fatalf("AtomV3 supply conversion: %v", err)
	}

	if supply != 25_500_000_000_000 {
		t.Fatalf(
			"AtomV3 supply units=%d want=25500000000000",
			supply,
		)
	}
}

func TestProtocolAmountRuntimeContractV1(t *testing.T) {
	if err := ValidateProtocolAmountRuntimeV1(
		ProtocolLegacyV1,
	); err != nil {
		t.Fatalf(
			"LegacyV1 runtime unexpectedly rejected: %v",
			err,
		)
	}

	err := ValidateProtocolAmountRuntimeV1(
		ProtocolAtomV3,
	)

	if strconv.IntSize == 64 {
		if err != nil {
			t.Fatalf(
				"64-bit AtomV3 runtime rejected: %v",
				err,
			)
		}
		return
	}

	if err == nil {
		t.Fatal(
			"non-64-bit AtomV3 runtime unexpectedly accepted",
		)
	}
}

func TestUnknownProtocolHasNoMonetarySemanticsV1(t *testing.T) {
	unknown := ProtocolVersion(
		"QC_UNKNOWN_PROTOCOL",
	)

	if _, err := MonetaryUnitAtProtocolV1(
		unknown,
	); err == nil {
		t.Fatal("unknown protocol monetary unit accepted")
	}

	if _, err := WholeQCAmountToProtocolUnitsV1(
		unknown,
		1,
	); err == nil {
		t.Fatal("unknown protocol amount conversion accepted")
	}
}
