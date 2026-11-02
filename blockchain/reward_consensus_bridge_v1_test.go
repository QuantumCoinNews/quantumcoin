package blockchain

import (
	"testing"

	"quantumcoin/wallet"
)

func TestValidateBlockRewardAgainstStateV1ConsensusBridgeBoundaries(
	t *testing.T,
) {
	tests := []struct {
		name              string
		height            int
		coinbaseAmount    int
		totalMintedBefore int
		totalSupply       int
		wantSubsidy       int
		wantFees          int
		wantTotal         int
		wantError         bool
	}{
		{
			name:              "genesis allocation within supply is accepted",
			height:            0,
			coinbaseAmount:    300,
			totalMintedBefore: 0,
			totalSupply:       1_000,
			wantSubsidy:       300,
			wantTotal:         300,
		},
		{
			name:              "genesis allocation above supply is rejected",
			height:            0,
			coinbaseAmount:    1_001,
			totalMintedBefore: 0,
			totalSupply:       1_000,
			wantError:         true,
		},
		{
			name:              "genesis legacy uncapped supply is preserved",
			height:            0,
			coinbaseAmount:    300,
			totalMintedBefore: 0,
			totalSupply:       0,
			wantSubsidy:       300,
			wantTotal:         300,
		},
		{
			name:              "final partial subsidy is accepted",
			height:            1,
			coinbaseAmount:    7,
			totalMintedBefore: 993,
			totalSupply:       1_000,
			wantSubsidy:       7,
			wantTotal:         7,
		},
		{
			name:              "final partial subsidy mismatch is rejected",
			height:            1,
			coinbaseAmount:    8,
			totalMintedBefore: 993,
			totalSupply:       1_000,
			wantError:         true,
		},
		{
			name:              "completed emission is rejected",
			height:            1,
			coinbaseAmount:    1,
			totalMintedBefore: 1_000,
			totalSupply:       1_000,
			wantError:         true,
		},
		{
			name:              "normal legacy uncapped subsidy is preserved",
			height:            1,
			coinbaseAmount:    50,
			totalMintedBefore: 0,
			totalSupply:       0,
			wantSubsidy:       50,
			wantTotal:         50,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chain, _, _, _ := newFeeConsensusTestStateV1(t)
			chain.TotalSupply = test.totalSupply

			miner := wallet.NewWallet()

			block := &Block{
				Index: test.height,
				Transactions: []*Transaction{
					feeTestCoinbaseV1(
						t,
						miner.GetAddress(),
						test.coinbaseAmount,
					),
				},
			}

			result, err := chain.ValidateBlockRewardAgainstStateV1(
				block,
				test.totalMintedBefore,
			)

			if test.wantError {
				if err == nil {
					t.Fatal("expected reward bridge validation error")
				}
				return
			}

			if err != nil {
				t.Fatalf(
					"unexpected reward bridge validation error: %v",
					err,
				)
			}

			if result.Subsidy != test.wantSubsidy ||
				result.Fees != test.wantFees ||
				result.CoinbaseTotal != test.wantTotal {
				t.Fatalf(
					"result=%+v, want subsidy=%d fees=%d total=%d",
					result,
					test.wantSubsidy,
					test.wantFees,
					test.wantTotal,
				)
			}
		})
	}
}
