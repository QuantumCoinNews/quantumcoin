package consensus

import "testing"

func TestLegacySubsidyAtHeight(t *testing.T) {
	interval := LegacyHalvingIntervalBlocks

	tests := []struct {
		name   string
		height uint64
		want   int64
	}{
		{
			name:   "genesis döneminde 50 QC",
			height: 0,
			want:   50,
		},
		{
			name:   "ilk halving öncesi 50 QC",
			height: interval - 1,
			want:   50,
		},
		{
			name:   "ilk halving sonrasi 25 QC",
			height: interval + 1,
			want:   25,
		},
		{
			name:   "ikinci halving sonrasi 12 QC",
			height: interval*2 + 1,
			want:   12,
		},
		{
			name:   "ucuncu halving sonrasi 6 QC",
			height: interval*3 + 1,
			want:   6,
		},
		{
			name:   "dorduncu halving sonrasi 3 QC",
			height: interval*4 + 1,
			want:   3,
		},
		{
			name:   "besinci halving sonrasi 1 QC",
			height: interval*5 + 1,
			want:   1,
		},
		{
			name:   "ödül 1 QC altına düşmez",
			height: interval * 100,
			want:   1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := LegacySubsidyAtHeight(test.height)
			if err != nil {
				t.Fatalf(
					"LegacySubsidyAtHeight(%d) error: %v",
					test.height,
					err,
				)
			}

			if got != test.want {
				t.Fatalf(
					"LegacySubsidyAtHeight(%d) = %d, want %d",
					test.height,
					got,
					test.want,
				)
			}
		})
	}
}

func TestSubsidyAtHeightRejectsInvalidParams(t *testing.T) {
	params := LegacyParamsV1()
	params.HalvingIntervalBlocks = 0

	_, err := SubsidyAtHeight(params, 0)
	if err == nil {
		t.Fatal("invalid params must be rejected")
	}
}

func TestSubsidyAtHeightDoesNotIncludeFees(t *testing.T) {
	got, err := LegacySubsidyAtHeight(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != LegacyInitialSubsidyQC {
		t.Fatalf(
			"subsidy = %d, want %d",
			got,
			LegacyInitialSubsidyQC,
		)
	}
}

func TestLegacySubsidyBoundaryCompatibilityV1(t *testing.T) {
	interval := LegacyHalvingIntervalBlocks

	tests := []struct {
		name   string
		height uint64
		want   int64
	}{
		{
			name:   "legacy son 50 QC blogu",
			height: interval,
			want:   50,
		},
		{
			name:   "legacy ilk 25 QC blogu",
			height: interval + 1,
			want:   25,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := LegacySubsidyAtHeight(test.height)
			if err != nil {
				t.Fatalf(
					"LegacySubsidyAtHeight(%d) error: %v",
					test.height,
					err,
				)
			}

			if got != test.want {
				t.Fatalf(
					"LegacySubsidyAtHeight(%d) = %d, want %d",
					test.height,
					got,
					test.want,
				)
			}
		})
	}
}

func TestCapSubsidyByRemainingSupplyV1(t *testing.T) {
	tests := []struct {
		name        string
		baseSubsidy int64
		totalMinted int64
		totalSupply int64
		want        int64
	}{
		{
			name:        "normal blok odulu korunur",
			baseSubsidy: 50,
			totalMinted: 100,
			totalSupply: 25_500_000,
			want:        50,
		},
		{
			name:        "son blok odulu kalan arza kisaltilir",
			baseSubsidy: 50,
			totalMinted: 25_499_975,
			totalSupply: 25_500_000,
			want:        25,
		},
		{
			name:        "arz tamamlandiysa odul sifirdir",
			baseSubsidy: 50,
			totalMinted: 25_500_000,
			totalSupply: 25_500_000,
			want:        0,
		},
		{
			name:        "basim arz sinirini asmissa odul sifirdir",
			baseSubsidy: 50,
			totalMinted: 25_500_001,
			totalSupply: 25_500_000,
			want:        0,
		},
		{
			name:        "legacy sinirsiz arz davranisi",
			baseSubsidy: 50,
			totalMinted: 100,
			totalSupply: 0,
			want:        50,
		},
		{
			name:        "gecersiz temel odul sifirdir",
			baseSubsidy: 0,
			totalMinted: 0,
			totalSupply: 25_500_000,
			want:        0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := CapSubsidyByRemainingSupply(
				test.baseSubsidy,
				test.totalMinted,
				test.totalSupply,
			)

			if got != test.want {
				t.Fatalf(
					"CapSubsidyByRemainingSupply(%d, %d, %d) = %d, want %d",
					test.baseSubsidy,
					test.totalMinted,
					test.totalSupply,
					got,
					test.want,
				)
			}
		})
	}
}
