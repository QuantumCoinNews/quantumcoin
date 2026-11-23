package consensus

import "testing"

func TestPostEmissionRewardOnlyVerifiedFees(t *testing.T) {
	for _, fee := range []int64{0, 3} {
		result, err := ValidateBlockRewardV1(100, fee, 0, fee, 1000)
		if err != nil || result.Subsidy != 0 || result.Fees != fee || result.CoinbaseTotal != fee {
			t.Fatalf("fees-only reward fee=%d result=%+v err=%v", fee, result, err)
		}
		if _, err := ValidateBlockRewardV1(100, fee+1, 0, fee, 1000); err == nil {
			t.Fatal("post-emission inflation accepted")
		}
		if fee > 0 {
			if _, err := ValidateBlockRewardV1(100, fee-1, 0, fee, 1000); err == nil {
				t.Fatal("incorrect fee total accepted")
			}
		}
	}
	if _, err := ValidateBlockRewardV1(100, 0, -1, 1, 1000); err == nil {
		t.Fatal("negative subsidy accepted")
	}
	if _, err := ValidateBlockRewardV1(100, -1, 0, -1, 1000); err == nil {
		t.Fatal("negative fee accepted")
	}
}

func TestPostEmissionCoinbaseLayoutKeepsGenesisAndNegativeGuards(t *testing.T) {
	for _, tc := range []struct {
		genesis bool
		amount  int64
		reject  bool
	}{{false, 0, false}, {true, 0, true}, {false, -1, true}} {
		_, err := ValidateCoinbaseLayoutV1([]*CoinbaseTransactionViewV1{{Coinbase: true, OutputAmounts: []int64{tc.amount}}}, tc.genesis)
		if (err != nil) != tc.reject {
			t.Fatalf("genesis=%v amount=%d err=%v", tc.genesis, tc.amount, err)
		}
	}
}
