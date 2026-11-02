package consensus

import (
	"math"
	"testing"
)

func TestValidateCoinbaseLayoutV1(t *testing.T) {
	tests := []struct {
		name         string
		genesis      bool
		transactions []*CoinbaseTransactionViewV1
		wantCount    int
		wantTotal    int64
		wantError    bool
	}{
		{
			name:    "normal blokta ilk ve tek coinbase",
			genesis: false,
			transactions: []*CoinbaseTransactionViewV1{
				{Coinbase: true, OutputAmounts: []int64{53}},
				{Coinbase: false},
			},
			wantCount: 1,
			wantTotal: 53,
		},
		{
			name:    "coinbase ilk islem degilse reddedilir",
			genesis: false,
			transactions: []*CoinbaseTransactionViewV1{
				{Coinbase: false},
				{Coinbase: true, OutputAmounts: []int64{50}},
			},
			wantError: true,
		},
		{
			name:    "normal blokta birden fazla coinbase reddedilir",
			genesis: false,
			transactions: []*CoinbaseTransactionViewV1{
				{Coinbase: true, OutputAmounts: []int64{25}},
				{Coinbase: true, OutputAmounts: []int64{25}},
			},
			wantError: true,
		},
		{
			name:    "genesis birden fazla tahsis islemini kabul eder",
			genesis: true,
			transactions: []*CoinbaseTransactionViewV1{
				{Coinbase: true, OutputAmounts: []int64{100}},
				{Coinbase: true, OutputAmounts: []int64{200}},
			},
			wantCount: 2,
			wantTotal: 300,
		},
		{
			name:         "bos blok reddedilir",
			genesis:      false,
			transactions: nil,
			wantError:    true,
		},
		{
			name:    "nil islem reddedilir",
			genesis: false,
			transactions: []*CoinbaseTransactionViewV1{
				nil,
			},
			wantError: true,
		},
		{
			name:    "coinbase olmayan blok reddedilir",
			genesis: false,
			transactions: []*CoinbaseTransactionViewV1{
				{Coinbase: false},
			},
			wantError: true,
		},
		{
			name:    "pozitif olmayan output reddedilir",
			genesis: false,
			transactions: []*CoinbaseTransactionViewV1{
				{Coinbase: true, OutputAmounts: []int64{0}},
			},
			wantError: true,
		},
		{
			name:    "coinbase toplami tasarsa reddedilir",
			genesis: true,
			transactions: []*CoinbaseTransactionViewV1{
				{Coinbase: true, OutputAmounts: []int64{math.MaxInt64}},
				{Coinbase: true, OutputAmounts: []int64{1}},
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ValidateCoinbaseLayoutV1(
				test.transactions,
				test.genesis,
			)

			if test.wantError {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}

			if result.Count != test.wantCount ||
				result.Total != test.wantTotal {
				t.Fatalf(
					"result = %+v, want count=%d total=%d",
					result,
					test.wantCount,
					test.wantTotal,
				)
			}
		})
	}
}
