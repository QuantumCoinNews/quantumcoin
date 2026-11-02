package blockchain

import (
	"sync"
	"testing"
)

func makeBlockPageSnapshotChain(
	count int,
) *Blockchain {
	blocks :=
		make(
			[]*Block,
			0,
			count,
		)

	for index := 0; index < count; index++ {
		blocks = append(
			blocks,
			&Block{
				Index: index,
				Hash: []byte{
					byte(index + 1),
				},
			},
		)
	}

	return &Blockchain{
		Blocks: blocks,
	}
}

func TestBlocksPageFromTipSnapshotPagination(
	t *testing.T,
) {
	chain :=
		makeBlockPageSnapshotChain(
			10,
		)

	first, total :=
		chain.BlocksPageFromTipSnapshot(
			0,
			3,
		)

	if total != 10 {
		t.Fatalf(
			"total=%d want=10",
			total,
		)
	}

	if len(first) != 3 ||
		first[0].Index != 7 ||
		first[1].Index != 8 ||
		first[2].Index != 9 {
		t.Fatalf(
			"first page=%v",
			blockIndexesForPageTest(first),
		)
	}

	second, total :=
		chain.BlocksPageFromTipSnapshot(
			3,
			3,
		)

	if total != 10 {
		t.Fatalf(
			"second total=%d want=10",
			total,
		)
	}

	if len(second) != 3 ||
		second[0].Index != 4 ||
		second[1].Index != 5 ||
		second[2].Index != 6 {
		t.Fatalf(
			"second page=%v",
			blockIndexesForPageTest(second),
		)
	}

	oldest, _ :=
		chain.BlocksPageFromTipSnapshot(
			9,
			3,
		)

	if len(oldest) != 1 ||
		oldest[0].Index != 0 {
		t.Fatalf(
			"oldest page=%v",
			blockIndexesForPageTest(oldest),
		)
	}

	empty, total :=
		chain.BlocksPageFromTipSnapshot(
			10,
			3,
		)

	if total != 10 ||
		len(empty) != 0 {
		t.Fatalf(
			"empty len=%d total=%d",
			len(empty),
			total,
		)
	}
}

func TestBlocksPageFromTipSnapshotIsDefensive(
	t *testing.T,
) {
	chain :=
		makeBlockPageSnapshotChain(
			4,
		)

	page, _ :=
		chain.BlocksPageFromTipSnapshot(
			0,
			2,
		)

	if len(page) != 2 {
		t.Fatalf(
			"page len=%d want=2",
			len(page),
		)
	}

	page[0].Hash[0] = 0xff

	chain.stateMu.RLock()
	defer chain.stateMu.RUnlock()

	if chain.Blocks[2].Hash[0] == 0xff {
		t.Fatal(
			"block page exposed live chain alias",
		)
	}
}

func TestBlocksPageFromTipSnapshotConcurrentWriter(
	t *testing.T,
) {
	chain :=
		makeBlockPageSnapshotChain(
			20,
		)

	const iterations = 500

	var wait sync.WaitGroup
	wait.Add(2)

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			chain.stateMu.Lock()

			height :=
				len(chain.Blocks)

			chain.Blocks = append(
				chain.Blocks,
				&Block{
					Index: height,
					Hash: []byte{
						byte(height),
					},
				},
			)

			chain.stateMu.Unlock()
		}
	}()

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			page, total :=
				chain.BlocksPageFromTipSnapshot(
					index%7,
					20,
				)

			if total < len(page) {
				t.Errorf(
					"total=%d page=%d",
					total,
					len(page),
				)
				return
			}
		}
	}()

	wait.Wait()
}

func blockIndexesForPageTest(
	blocks []*Block,
) []int {
	result :=
		make(
			[]int,
			0,
			len(blocks),
		)

	for _, block := range blocks {
		if block == nil {
			result = append(
				result,
				-1,
			)
			continue
		}

		result = append(
			result,
			block.Index,
		)
	}

	return result
}
