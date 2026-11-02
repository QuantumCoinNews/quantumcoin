package internal

import (
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"quantumcoin/blockchain"
)

// newReindexTestChain builds a minimal chain with:
//   - txA (coinbase-shaped, no inputs): 3 outputs at index 0, 1, 2
//   - spendTx: spends txA's output at index 1 (the middle one)
//
// This is the exact shape that exposes QMA-0043: if rebuilding removed the
// spent slot instead of preserving it, output index 2 would shift to slot 1.
func newReindexTestChain() *blockchain.Blockchain {
	txA := &blockchain.Transaction{
		ID: []byte{0xAA},
		Outputs: []blockchain.TransactionOutput{
			{Amount: 10, PubKeyHash: []byte{0x01}},
			{Amount: 20, PubKeyHash: []byte{0x02}},
			{Amount: 30, PubKeyHash: []byte{0x03}},
		},
	}

	spendTx := &blockchain.Transaction{
		ID: []byte{0xBB},
		Inputs: []blockchain.TransactionInput{
			{
				TxID:      []byte{0xAA},
				OutIndex:  1,
				Signature: []byte{0x01},
				PubKey:    []byte{0x02},
			},
		},
		Outputs: []blockchain.TransactionOutput{
			{Amount: 20, PubKeyHash: []byte{0x99}},
		},
	}

	block := &blockchain.Block{
		Index:        0,
		Hash:         []byte{0x01},
		PrevHash:     []byte{0x00},
		Transactions: []*blockchain.Transaction{txA, spendTx},
	}

	return &blockchain.Blockchain{
		Blocks: []*blockchain.Block{block},
	}
}

func TestReindexPreservesOutIndexAfterPartialSpendV1(t *testing.T) {
	chain := newReindexTestChain()
	set := NewUTXOSet()
	set.Reindex(chain)

	txID := hex.EncodeToString([]byte{0xAA})
	outs, ok := set.Snapshot()[txID]
	if !ok {
		t.Fatalf("expected UTXO entry for txID=%s", txID)
	}

	if len(outs) != 3 {
		t.Fatalf(
			"outs length=%d want=3 (spent slot must be kept, not removed)",
			len(outs),
		)
	}

	if outs[0].Amount != 10 {
		t.Fatalf("outs[0].Amount=%d want=10", outs[0].Amount)
	}

	if outs[1].Amount != 0 {
		t.Fatalf("outs[1].Amount=%d want=0", outs[1].Amount)
	}

	if outs[2].Amount != 30 {
		t.Fatalf(
			"outs[2].Amount=%d want=30 (output must remain at OutIndex 2)",
			outs[2].Amount,
		)
	}
}

func TestFindSpendableOutputsReportsRealOutIndexV1(t *testing.T) {
	chain := newReindexTestChain()
	set := NewUTXOSet()
	set.Reindex(chain)

	spendable, total := set.FindSpendableOutputs([]byte{0x03}, 1)

	txID := hex.EncodeToString([]byte{0xAA})
	indices, ok := spendable[txID]
	if !ok || len(indices) != 1 {
		t.Fatalf("expected one spendable index for txID=%s, got=%v", txID, spendable)
	}

	if indices[0] != 2 {
		t.Fatalf("reported OutIndex=%d want=2", indices[0])
	}

	if total != 30 {
		t.Fatalf("total=%d want=30", total)
	}
}

func TestFindSpendableOutputsAndFindUTXOsSkipSpentSlotV1(t *testing.T) {
	chain := newReindexTestChain()
	set := NewUTXOSet()
	set.Reindex(chain)

	if utxos := set.FindUTXOs([]byte{0x02}); len(utxos) != 0 {
		t.Fatalf("spent output returned as live UTXO: %+v", utxos)
	}

	spendable, total := set.FindSpendableOutputs([]byte{0x02}, 1)
	if len(spendable) != 0 || total != 0 {
		t.Fatalf("spent output reported spendable: %v total=%d", spendable, total)
	}

	if utxos := set.FindUTXOs(nil); len(utxos) != 0 {
		t.Fatalf("nil pubKeyHash query returned UTXO: %+v", utxos)
	}
}

func TestReindexSourceDoesNotShiftIndicesV1(t *testing.T) {
	source, err := os.ReadFile("utxo.go")
	if err != nil {
		t.Fatalf("read utxo.go: %v", err)
	}

	text := string(source)

	forbidden := []string{
		"outs[:idx], outs[idx+1:]",
		"UTXOs map[string]",
		"set.UTXOs",
		"AllBlocksSnapshot",
	}
	for _, marker := range forbidden {
		if strings.Contains(text, marker) {
			t.Fatalf("QMA-0043 unsafe UTXO marker reappeared: %s", marker)
		}
	}

	required := []string{
		"canonical.UpdateUTXOSetSynchronized()",
		"canonical.FindSpendableOutputsSnapshot(",
		"canonical.FindUTXOsSnapshot(",
		"canonical.StateSnapshot()",
	}
	for _, marker := range required {
		if !strings.Contains(text, marker) {
			t.Fatalf("canonical index-preserving marker missing: %s", marker)
		}
	}
}
