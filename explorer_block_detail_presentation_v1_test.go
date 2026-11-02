package main

import (
	"os"
	"strings"
	"testing"

	"quantumcoin/blockchain"
)

func TestBlockDetailDoesNotExposeCanonicalBlockDirectlyV1(
	t *testing.T,
) {
	sourceBytes, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	source := string(sourceBytes)

	start := strings.Index(
		source,
		"func handleBlockDetail(",
	)
	if start < 0 {
		t.Fatal("handleBlockDetail not found")
	}

	tail := source[start:]

	end := strings.Index(
		tail,
		"/* ---------- Burn & stake stubs ---------- */",
	)
	if end < 0 {
		t.Fatal("handleBlockDetail end boundary not found")
	}

	handler := tail[:end]

	if strings.Contains(
		handler,
		"writeOK(w, blk)",
	) {
		t.Fatal(
			"block detail exposes canonical block directly; " +
				"AtomV3 transaction/output amounts remain raw protocol units",
		)
	}

	if got := strings.Count(
		handler,
		"newBlockDetailPresentationV1(bc, blk)",
	); got != 2 {
		t.Fatalf(
			"presentation calls=%d want=2 (index + hash)",
			got,
		)
	}
}

type explorerAmountConverterStubV1 struct{}

func (explorerAmountConverterStubV1) TransactionAmountQCAtHeightV1(
	amount float64,
	height int,
) float64 {
	if height >= 1 {
		return amount / 1_000_000
	}

	return amount
}

func TestBlockDetailPresentationUsesContainingBlockHeightV1(
	t *testing.T,
) {
	canonical := &blockchain.Block{
		Index: 1,
		Transactions: []*blockchain.Transaction{
			{
				Amount: 1,
				Outputs: []blockchain.TransactionOutput{
					{
						Amount: 1,
					},
					{
						Amount: 50_000_000,
					},
				},
			},
		},
	}

	view :=
		newBlockDetailPresentationV1(
			explorerAmountConverterStubV1{},
			canonical,
		)

	if view == nil {
		t.Fatal("nil explorer presentation")
	}

	if len(view.Transactions) != 1 {
		t.Fatalf(
			"transactions=%d want=1",
			len(view.Transactions),
		)
	}

	tx := view.Transactions[0]
	if tx == nil {
		t.Fatal("nil transaction presentation")
	}

	if tx.Amount != 0.000001 {
		t.Fatalf(
			"transaction amount=%v want=0.000001 QC",
			tx.Amount,
		)
	}

	if len(tx.Outputs) != 2 {
		t.Fatalf(
			"outputs=%d want=2",
			len(tx.Outputs),
		)
	}

	if tx.Outputs[0].Amount != 0.000001 {
		t.Fatalf(
			"one atom output=%v want=0.000001 QC",
			tx.Outputs[0].Amount,
		)
	}

	if tx.Outputs[1].Amount != 50 {
		t.Fatalf(
			"50 QC output=%v want=50 QC",
			tx.Outputs[1].Amount,
		)
	}

	// Presentation must never mutate canonical consensus data.
	if canonical.Transactions[0].Amount != 1 {
		t.Fatalf(
			"canonical transaction mutated=%v want=1 atom",
			canonical.Transactions[0].Amount,
		)
	}

	if canonical.Transactions[0].Outputs[0].Amount != 1 {
		t.Fatalf(
			"canonical output mutated=%d want=1 atom",
			canonical.Transactions[0].Outputs[0].Amount,
		)
	}
}

func TestBlockDetailPresentationPreservesLegacyQCUnitsV1(
	t *testing.T,
) {
	canonical := &blockchain.Block{
		Index: 0,
		Transactions: []*blockchain.Transaction{
			{
				Amount: 50,
				Outputs: []blockchain.TransactionOutput{
					{
						Amount: 50,
					},
				},
			},
		},
	}

	view :=
		newBlockDetailPresentationV1(
			explorerAmountConverterStubV1{},
			canonical,
		)

	if view.Transactions[0].Amount != 50 {
		t.Fatalf(
			"legacy transaction=%v want=50 QC",
			view.Transactions[0].Amount,
		)
	}

	if view.Transactions[0].Outputs[0].Amount != 50 {
		t.Fatalf(
			"legacy output=%v want=50 QC",
			view.Transactions[0].Outputs[0].Amount,
		)
	}
}
