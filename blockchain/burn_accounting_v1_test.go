package blockchain

import "testing"

// QMA-0062:
// Burn accounting must be derived from the current canonical chain.
// It must not mutate the configured TotalSupply hard-cap authority.
func TestCanonicalTotalBurnedAtomsV1(t *testing.T) {
	bc := &Blockchain{
		TotalSupply: 25_500_000,
		Blocks: []*Block{
			{
				Index: 0,
				Transactions: []*Transaction{
					{
						Outputs: []TransactionOutput{
							NewCanonicalBurnOutputV1(1),
						},
					},
				},
			},
			{
				Index: 1,
				Transactions: []*Transaction{
					{
						Outputs: []TransactionOutput{
							NewCanonicalBurnOutputV1(9),
							{
								Amount:     100,
								PubKeyHash: []byte("normal-output"),
							},
						},
					},
				},
			},
		},
	}

	if got := bc.TotalBurnedAtomsV1(); got != 10_000_000 {
		t.Fatalf(
			"total burned atoms=%d want=10000000",
			got,
		)
	}

	if bc.TotalSupply != 25_500_000 {
		t.Fatalf(
			"burn accounting changed hard cap: got=%d want=25500000",
			bc.TotalSupply,
		)
	}
}

// QMA-0062:
// Burn accounting must follow the current canonical chain. If a block
// disappears from bc.Blocks after a reorg/replacement, its burn must
// disappear from the derived total automatically.
func TestCanonicalTotalBurnedAtomsFollowsCanonicalChainV1(t *testing.T) {
	genesis := &Block{
		Index: 0,
		Transactions: []*Transaction{
			{
				Outputs: []TransactionOutput{
					NewCanonicalBurnOutputV1(2),
				},
			},
		},
	}

	oldTip := &Block{
		Index: 1,
		Transactions: []*Transaction{
			{
				Outputs: []TransactionOutput{
					NewCanonicalBurnOutputV1(8),
				},
			},
		},
	}

	bc := &Blockchain{
		TotalSupply: 25_500_000,
		Blocks: []*Block{
			genesis,
			oldTip,
		},
	}

	if got := bc.TotalBurnedAtomsV1(); got != 10_000_000 {
		t.Fatalf(
			"pre-reorg total burned atoms=%d want=10000000",
			got,
		)
	}

	// Simulate the canonical-chain view after replacement.
	// The orphaned oldTip burn must no longer count.
	bc.Blocks = []*Block{
		genesis,
		{
			Index:        1,
			Transactions: nil,
		},
	}

	if got := bc.TotalBurnedAtomsV1(); got != 2_000_000 {
		t.Fatalf(
			"post-reorg total burned atoms=%d want=2000000",
			got,
		)
	}

	if bc.TotalSupply != 25_500_000 {
		t.Fatalf(
			"reorg burn accounting changed hard cap: got=%d want=25500000",
			bc.TotalSupply,
		)
	}
}
