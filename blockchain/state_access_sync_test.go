package blockchain

import (
	"sync"
	"testing"
)

func TestSynchronizedStateFacadeWithConcurrentWriter(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	const iterations = 750

	var wait sync.WaitGroup
	wait.Add(2)

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			chain.stateMu.Lock()
			chain.TotalSupply = 25_500_000 + index
			chain.stateMu.Unlock()
		}
	}()

	go func() {
		defer wait.Done()

		for index := 0; index < iterations; index++ {
			_ = chain.TotalSupplySnapshot()
			_ = chain.HeightSnapshot()
			_ = chain.TotalMintedSnapshot()
			_ = chain.NextDifficultySnapshot(1)
			_ = chain.ProtocolAtHeightSnapshotV1(0)
		}
	}()

	wait.Wait()
}

func TestExternalStateFacadeReturnsDetachedTip(
	t *testing.T,
) {
	chain := newStateSnapshotTestChain()

	tip := chain.TipSnapshot()
	if tip == nil {
		t.Fatal("tip snapshot is nil")
	}

	tip.Hash[0] = 0xff

	if chain.Blocks[0].Hash[0] != 0x31 {
		t.Fatal("external state facade exposed live tip alias")
	}
}

func TestFindSpendableOutputsSnapshotNilBlockchain(
	t *testing.T,
) {
	var chain *Blockchain

	outputs, accumulated :=
		chain.FindSpendableOutputsSnapshot(
			nil,
			1,
		)

	if outputs != nil {
		t.Fatalf(
			"outputs=%v want=nil",
			outputs,
		)
	}

	if accumulated != 0 {
		t.Fatalf(
			"accumulated=%d want=0",
			accumulated,
		)
	}
}
