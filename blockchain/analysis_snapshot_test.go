package blockchain

import (
	"bytes"
	"testing"
)

func TestAdvisorySnapshotsBoundBeforeCloneAndRemainDetached(t *testing.T) {
	chain := NewBlockchain(1, 0)
	tx := &Transaction{ID: bytes.Repeat([]byte{1}, 32), Sender: "sender", Outputs: []TransactionOutput{{Amount: 1, PubKeyHash: bytes.Repeat([]byte{2}, 20)}}}
	for i := 0; i < AnalysisMaxBlocks+10; i++ {
		chain.Blocks = append(chain.Blocks, &Block{Index: i + 1, Transactions: []*Transaction{tx}})
	}
	blocks, truncated := chain.AnalysisBlocksSnapshot()
	if !truncated || len(blocks) > AnalysisMaxBlocks {
		t.Fatal("chain scan unbounded")
	}
	blocks[0].Transactions[0].Outputs[0].PubKeyHash[0] = 99
	if tx.Outputs[0].PubKeyHash[0] != 2 {
		t.Fatal("snapshot aliases chain")
	}
	oversized := &Transaction{Inputs: make([]TransactionInput, 10000)}
	chain.pendingTxs = make([]*Transaction, AnalysisMaxTransactions+10)
	for i := range chain.pendingTxs {
		chain.pendingTxs[i] = tx
	}
	chain.pendingTxs[0] = oversized
	pending, _, truncated := chain.AnalysisPendingSnapshot()
	if !truncated || len(pending) >= AnalysisMaxTransactions {
		t.Fatal("oversized transaction cloned or pending unbounded")
	}
}
func TestBlockSummaryPageCountsWithoutCloningTransactions(t *testing.T) {
	chain := NewBlockchain(1, 0)
	chain.Blocks = append(chain.Blocks, &Block{Index: 1, Transactions: make([]*Transaction, 1000000), Hash: bytes.Repeat([]byte{3}, 32)})
	page, total := chain.BlockSummaryPageFromTipSnapshot(0, int(^uint(0)>>1))
	if total != 2 || len(page) != 2 || page[1].TxCount != 1000000 {
		t.Fatal("summary pagination incorrect")
	}
	page[1].Hash[0] = 99
	if chain.Blocks[1].Hash[0] != 3 {
		t.Fatal("summary aliases chain")
	}
}
