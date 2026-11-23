package blockchain

import (
	"errors"
	"testing"
	"time"
)

func TestCanonicalBlockViewExcludesWritersUntilLedgerCommitReturns(t *testing.T) {
	bc := &Blockchain{Blocks: []*Block{{Index: 0, Hash: []byte{1}}, {Index: 1, Hash: []byte{2}, Miner: "original"}}}
	writing, done := make(chan struct{}), make(chan struct{})
	err := bc.WithCanonicalBlockViewV1([]int{1}, func(view CanonicalBlockViewV1) error {
		if view.TipHeight != 1 || view.Blocks[1].Hash != "02" {
			t.Fatal("incorrect coherent view")
		}
		view.Blocks[1] = CanonicalBlockIdentityV1{Hash: "forged", Miner: "forged"}
		go func() {
			close(writing)
			bc.stateMu.Lock()
			bc.Blocks[1].Miner = "replacement"
			bc.stateMu.Unlock()
			close(done)
		}()
		<-writing
		select {
		case <-done:
			t.Error("chain changed during dependent ledger commit")
		case <-time.After(30 * time.Millisecond):
		}
		if bc.Blocks[1].Miner != "original" || bc.Blocks[1].Hash[0] != 2 {
			t.Error("view modified live state")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("chain writer did not resume after commit")
	}
}

func TestCanonicalBlockViewRejectsMissingContextAndPropagatesErrors(t *testing.T) {
	var absent *Blockchain
	if err := absent.WithCanonicalBlockViewV1(nil, func(CanonicalBlockViewV1) error { t.Fatal("nil chain invoked visitor"); return nil }); err == nil {
		t.Fatal("nil chain accepted")
	}
	bc := new(Blockchain)
	if err := bc.WithCanonicalBlockViewV1(nil, nil); err == nil {
		t.Fatal("nil visitor accepted")
	}
	want := errors.New("dependent ledger failed")
	if err := bc.WithCanonicalBlockViewV1(nil, func(view CanonicalBlockViewV1) error {
		if view.TipHeight != -1 {
			t.Error("empty chain has a tip")
		}
		return want
	}); !errors.Is(err, want) {
		t.Fatalf("lost ledger error: %v", err)
	}
}
