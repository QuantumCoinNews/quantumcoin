package blockchain

import (
	"sync"
	"testing"
)

func TestCanonicalStateReadersSynchronizeWithWriter(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	var wait sync.WaitGroup

	const iterations = 500

	wait.Add(2)

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			chain.stateMu.Lock()

			chain.TotalSupply =
				25_500_000 + index

			chain.Blocks = append(
				chain.Blocks,
				&Block{
					Index: index + 1,
					Hash: []byte{
						byte(index),
						byte(index >> 8),
					},
				},
			)

			chain.pendingTxs = append(
				chain.pendingTxs,
				&Transaction{
					ID: []byte{
						byte(index),
					},
				},
			)

			chain.stateMu.Unlock()
		}
	}()

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			_ = chain.BlockCountSnapshot()
			_ = chain.TotalSupplySnapshot()
			_ = chain.BestHeightSnapshot()
			_ = chain.TipSnapshot()
			_ = chain.ChainSummarySnapshot()

			blocks, total :=
				chain.BlocksRangeSnapshot(
					0,
					4,
				)

			if len(blocks) > 4 {
				t.Errorf(
					"bounded blocks len=%d",
					len(blocks),
				)
				return
			}

			if total < len(blocks) {
				t.Errorf(
					"total=%d len=%d",
					total,
					len(blocks),
				)
				return
			}

			pending, pendingTotal :=
				chain.PendingTransactionsRangeSnapshot(
					0,
					4,
				)

			if len(pending) > 4 {
				t.Errorf(
					"bounded pending len=%d",
					len(pending),
				)
				return
			}

			if pendingTotal < len(pending) {
				t.Errorf(
					"pending total=%d len=%d",
					pendingTotal,
					len(pending),
				)
				return
			}
		}
	}()

	wait.Wait()
}

func TestBlockLookupSnapshotsAreDefensive(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	byIndex := chain.BlockByIndexSnapshot(
		0,
	)

	if byIndex == nil {
		t.Fatal(
			"block-by-index snapshot is nil",
		)
	}

	byHash := chain.BlockByHashSnapshot(
		[]byte{0x31},
	)

	if byHash == nil {
		t.Fatal(
			"block-by-hash snapshot is nil",
		)
	}

	byIndex.Hash[0] = 0xff
	byHash.Metadata["source"] = "attacker"

	if chain.Blocks[0].Hash[0] != 0x31 {
		t.Fatal(
			"block-by-index snapshot aliased live hash",
		)
	}

	if chain.Blocks[0].
		Metadata["source"] !=
		"state-snapshot-test" {
		t.Fatal(
			"block-by-hash snapshot aliased live metadata",
		)
	}
}

func TestAllBlocksSnapshotIsDefensive(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	blocks := chain.AllBlocksSnapshot()

	if len(blocks) != 1 {
		t.Fatalf(
			"len=%d want=1",
			len(blocks),
		)
	}

	blocks[0].Hash[0] = 0xff
	blocks[0].
		Transactions[0].
		Outputs[0].
		PubKeyHash[0] = 0xff

	if chain.Blocks[0].Hash[0] != 0x31 {
		t.Fatal(
			"all-block snapshot aliased live block",
		)
	}

	if chain.Blocks[0].
		Transactions[0].
		Outputs[0].
		PubKeyHash[0] != 0x11 {
		t.Fatal(
			"all-block snapshot aliased live transaction",
		)
	}
}
