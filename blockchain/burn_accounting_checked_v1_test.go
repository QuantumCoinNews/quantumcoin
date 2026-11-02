package blockchain

import (
	"math"
	"testing"
)

// QMA-0062:
// Burn accounting is consensus/economy accounting. Integer overflow must
// fail closed instead of silently saturating or wrapping.
func TestCanonicalTotalBurnedAtomsRejectsOverflowV1(t *testing.T) {
	bc := &Blockchain{
		TotalSupply: 25_500_000,
		Blocks: []*Block{
			{
				Index: 0,
				Transactions: []*Transaction{
					{
						Outputs: []TransactionOutput{
							NewCanonicalBurnOutputV1(math.MaxInt),
							NewCanonicalBurnOutputV1(math.MaxInt),
						},
					},
				},
			},
		},
	}

	_, err := bc.TotalBurnedAtomsCheckedV1()
	if err == nil {
		t.Fatal("burn accounting overflow unexpectedly accepted")
	}

	if bc.TotalSupply != 25_500_000 {
		t.Fatalf(
			"burn accounting changed hard cap: got=%d want=25500000",
			bc.TotalSupply,
		)
	}
}
