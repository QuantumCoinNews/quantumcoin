package consensus

import "testing"

func TestCanonicalQCDenominationV1(t *testing.T) {
	tests := []struct {
		raw   string
		atoms int64
	}{
		{"0", 0},
		{"1", 1_000_000},
		{"50", 50_000_000},
		{"0.000001", 1},
		{"0.00001", 10},
		{"1.25", 1_250_000},
		{"1.250001", 1_250_001},
		{"25500000", 25_500_000_000_000},
	}

	for _, test := range tests {
		got, err := ParseQCToAtomsV1(test.raw)
		if err != nil {
			t.Fatalf(
				"ParseQCToAtomsV1(%q): %v",
				test.raw,
				err,
			)
		}

		if got != test.atoms {
			t.Fatalf(
				"ParseQCToAtomsV1(%q)=%d want=%d",
				test.raw,
				got,
				test.atoms,
			)
		}
	}
}

func TestCanonicalQCFormattingV1(t *testing.T) {
	tests := []struct {
		atoms int64
		qc    string
	}{
		{0, "0"},
		{1, "0.000001"},
		{10, "0.00001"},
		{1_000_000, "1"},
		{50_000_000, "50"},
		{1_250_001, "1.250001"},
		{25_500_000_000_000, "25500000"},
	}

	for _, test := range tests {
		got := FormatAtomsAsQCV1(test.atoms)
		if got != test.qc {
			t.Fatalf(
				"FormatAtomsAsQCV1(%d)=%q want=%q",
				test.atoms,
				got,
				test.qc,
			)
		}
	}
}

func TestCanonicalQCParserRejectsInvalidPrecisionV1(t *testing.T) {
	invalid := []string{
		"",
		"-1",
		"0.0000001",
		"1.0000001",
		"1.2.3",
		"abc",
	}

	for _, raw := range invalid {
		if _, err := ParseQCToAtomsV1(raw); err == nil {
			t.Fatalf(
				"ParseQCToAtomsV1(%q) unexpectedly succeeded",
				raw,
			)
		}
	}
}

func TestWholeQCToAtomsV1PreservesEconomicValues(t *testing.T) {
	rewardAtoms, err := WholeQCToAtomsV1(
		CanonicalInitialMiningRewardQCV1,
	)
	if err != nil {
		t.Fatalf("reward conversion: %v", err)
	}

	if rewardAtoms != 50_000_000 {
		t.Fatalf(
			"initial reward atoms=%d want=50000000",
			rewardAtoms,
		)
	}

	supplyAtoms, err := WholeQCToAtomsV1(
		CanonicalTotalSupplyQCV1,
	)
	if err != nil {
		t.Fatalf("supply conversion: %v", err)
	}

	if supplyAtoms != 25_500_000_000_000 {
		t.Fatalf(
			"total supply atoms=%d want=25500000000000",
			supplyAtoms,
		)
	}
}
