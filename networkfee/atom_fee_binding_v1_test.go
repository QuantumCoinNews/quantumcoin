package networkfee

import (
	"testing"

	qconsensus "quantumcoin/consensus"
)

func TestQMA0060NetworkFeePolicyBindsCanonicalAtomsV1(t *testing.T) {
	if DefaultMicroFeeTargetAtoms != 1 {
		t.Fatalf(
			"base fee atoms=%d want=1",
			DefaultMicroFeeTargetAtoms,
		)
	}

	if DefaultCongestionCeilingAtoms != 10 {
		t.Fatalf(
			"congestion ceiling atoms=%d want=10",
			DefaultCongestionCeilingAtoms,
		)
	}

	baseAtoms, err := qconsensus.ParseQCToAtomsV1(
		DefaultMicroFeeTargetQC,
	)
	if err != nil {
		t.Fatalf("parse base fee: %v", err)
	}

	if baseAtoms != DefaultMicroFeeTargetAtoms {
		t.Fatalf(
			"base fee=%q parsed=%d want=%d",
			DefaultMicroFeeTargetQC,
			baseAtoms,
			DefaultMicroFeeTargetAtoms,
		)
	}

	ceilingAtoms, err := qconsensus.ParseQCToAtomsV1(
		DefaultCongestionCeilingQC,
	)
	if err != nil {
		t.Fatalf("parse congestion ceiling: %v", err)
	}

	if ceilingAtoms != DefaultCongestionCeilingAtoms {
		t.Fatalf(
			"ceiling=%q parsed=%d want=%d",
			DefaultCongestionCeilingQC,
			ceilingAtoms,
			DefaultCongestionCeilingAtoms,
		)
	}
}

func TestQMA0060NetworkFeeDefaultPolicyStillValidV1(t *testing.T) {
	policy := DefaultPolicyV1()

	if err := ValidatePolicyV1(policy); err != nil {
		t.Fatalf(
			"default network fee policy invalid: %v",
			err,
		)
	}
}

func TestQMA0060CanonicalFeeAtomScaleMatchesQCScaleV1(t *testing.T) {
	if qconsensus.CanonicalAtomsPerQCV1 != 1_000_000 {
		t.Fatalf(
			"atoms per QC=%d want=1000000",
			qconsensus.CanonicalAtomsPerQCV1,
		)
	}

	if DefaultMicroFeeTargetAtoms != 1 {
		t.Fatalf(
			"micro fee atoms=%d want=1",
			DefaultMicroFeeTargetAtoms,
		)
	}

	if DefaultCongestionCeilingAtoms != 10 {
		t.Fatalf(
			"ceiling atoms=%d want=10",
			DefaultCongestionCeilingAtoms,
		)
	}
}
