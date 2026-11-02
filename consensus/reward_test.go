package consensus

import (
	"math"
	"testing"
)

func TestValidateBlockRewardV1(t *testing.T) {
	tests := []struct {
		name          string
		height        uint64
		coinbaseTotal int64
		subsidy       int64
		fees          int64
		totalSupply   int64
		wantSubsidy   int64
		wantFees      int64
		wantTotal     int64
		wantError     bool
	}{
		{
			name:          "normal block accepts subsidy plus verified fees",
			height:        1,
			coinbaseTotal: 53,
			subsidy:       50,
			fees:          3,
			totalSupply:   1_000,
			wantSubsidy:   50,
			wantFees:      3,
			wantTotal:     53,
		},
		{
			name:          "normal block rejects coinbase mismatch",
			height:        1,
			coinbaseTotal: 54,
			subsidy:       50,
			fees:          3,
			totalSupply:   1_000,
			wantError:     true,
		},
		{
			name:          "normal block rejects completed emission",
			height:        100,
			coinbaseTotal: 3,
			subsidy:       0,
			fees:          3,
			totalSupply:   1_000,
			wantError:     true,
		},
		{
			name:          "normal block rejects subsidy plus fee overflow",
			height:        1,
			coinbaseTotal: math.MaxInt64,
			subsidy:       math.MaxInt64,
			fees:          1,
			totalSupply:   0,
			wantError:     true,
		},
		{
			name:          "genesis accepts allocation within supply",
			height:        0,
			coinbaseTotal: 300,
			totalSupply:   1_000,
			wantSubsidy:   300,
			wantTotal:     300,
		},
		{
			name:          "genesis rejects allocation above supply",
			height:        0,
			coinbaseTotal: 1_001,
			totalSupply:   1_000,
			wantError:     true,
		},
		{
			name:          "genesis preserves legacy uncapped supply mode",
			height:        0,
			coinbaseTotal: 300,
			totalSupply:   0,
			wantSubsidy:   300,
			wantTotal:     300,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ValidateBlockRewardV1(
				test.height,
				test.coinbaseTotal,
				test.subsidy,
				test.fees,
				test.totalSupply,
			)

			if test.wantError {
				if err == nil {
					t.Fatal("expected reward validation error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected reward validation error: %v", err)
			}

			if result.Subsidy != test.wantSubsidy ||
				result.Fees != test.wantFees ||
				result.CoinbaseTotal != test.wantTotal {
				t.Fatalf(
					"result = %+v, want subsidy=%d fees=%d total=%d",
					result,
					test.wantSubsidy,
					test.wantFees,
					test.wantTotal,
				)
			}
		})
	}
}
