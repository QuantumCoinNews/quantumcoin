package internal

import (
	"os"
	"strings"
	"sync"
	"testing"

	"quantumcoin/blockchain"
)

func TestUTXOSetRequiresCanonicalBindingV1(t *testing.T) {
	set := NewUTXOSet()
	if set == nil {
		t.Fatal("NewUTXOSet returned nil")
	}

	if snapshot := set.Snapshot(); len(snapshot) != 0 {
		t.Fatalf("unbound snapshot is not empty: %+v", snapshot)
	}

	spendable, total := set.FindSpendableOutputs([]byte{1}, 1)
	if len(spendable) != 0 || total != 0 {
		t.Fatalf("unbound adapter returned spendable state: %v total=%d", spendable, total)
	}

	if utxos := set.FindUTXOs([]byte{1}); len(utxos) != 0 {
		t.Fatalf("unbound adapter returned UTXOs: %+v", utxos)
	}
}

func TestUTXOSetCannotSwitchCanonicalAuthorityV1(t *testing.T) {
	first := newReindexTestChain()
	second := newReindexTestChain()
	set := NewUTXOSet()

	if !set.BindCanonicalBlockchain(first) {
		t.Fatal("initial canonical binding rejected")
	}
	if !set.BindCanonicalBlockchain(first) {
		t.Fatal("idempotent canonical binding rejected")
	}
	if set.BindCanonicalBlockchain(second) {
		t.Fatal("adapter switched canonical Blockchain authority")
	}
	if set.BindCanonicalBlockchain(nil) {
		t.Fatal("nil canonical binding accepted")
	}

	set.Reindex(second)
	if set.canonicalSnapshot() != first {
		t.Fatal("Reindex switched canonical authority")
	}
}

func TestUTXOSetSnapshotIsDetachedV1(t *testing.T) {
	chain := newReindexTestChain()
	set := NewUTXOSet()
	set.Reindex(chain)

	first := set.Snapshot()
	outputs := first["aa"]
	if len(outputs) != 3 {
		t.Fatalf("snapshot outputs=%d want=3", len(outputs))
	}

	outputs[0].Amount = 999
	outputs[0].PubKeyHash[0] = 0xff
	first["attacker"] = []blockchain.TransactionOutput{{Amount: 999}}

	second := set.Snapshot()
	if second["aa"][0].Amount != 10 {
		t.Fatal("snapshot mutation changed canonical output amount")
	}
	if second["aa"][0].PubKeyHash[0] != 0x01 {
		t.Fatal("snapshot mutation changed canonical PubKeyHash")
	}
	if _, ok := second["attacker"]; ok {
		t.Fatal("snapshot map mutation reached canonical state")
	}
}

func TestUTXOSetDelegatesCoinbaseMaturityV1(t *testing.T) {
	chain := newReindexTestChain()
	chain.SetCoinbaseMaturity(2)

	set := NewUTXOSet()
	set.Reindex(chain)

	spendable, total := set.FindSpendableOutputs([]byte{0x03}, 1)
	if len(spendable) != 0 || total != 0 {
		t.Fatalf("immature coinbase output reported spendable: %v total=%d", spendable, total)
	}
	if outputs := set.FindUTXOs([]byte{0x03}); len(outputs) != 0 {
		t.Fatalf("immature coinbase output returned: %+v", outputs)
	}

	chain.Blocks = append(chain.Blocks, &blockchain.Block{Index: 1})
	set.Reindex(chain)

	spendable, total = set.FindSpendableOutputs([]byte{0x03}, 1)
	if len(spendable) != 1 || total != 30 {
		t.Fatalf("mature output missing: %v total=%d", spendable, total)
	}
	if outputs := set.FindUTXOs([]byte{0x03}); len(outputs) != 1 || outputs[0].Amount != 30 {
		t.Fatalf("mature output result=%+v", outputs)
	}
}

func TestUTXOSetNilReceiversAndInputsAreSafeV1(t *testing.T) {
	var set *UTXOSet
	set.Reindex(nil)

	if set.BindCanonicalBlockchain(newReindexTestChain()) {
		t.Fatal("nil receiver accepted canonical binding")
	}
	if snapshot := set.Snapshot(); len(snapshot) != 0 {
		t.Fatalf("nil receiver snapshot=%+v", snapshot)
	}
	if outputs := set.FindUTXOs(nil); len(outputs) != 0 {
		t.Fatalf("nil receiver UTXOs=%+v", outputs)
	}

	spendable, total := set.FindSpendableOutputs(nil, -1)
	if len(spendable) != 0 || total != 0 {
		t.Fatalf("nil receiver spendable=%v total=%d", spendable, total)
	}
}

func TestUTXOSetConcurrentCanonicalReadsV1(t *testing.T) {
	chain := newReindexTestChain()
	set := NewUTXOSet()
	set.Reindex(chain)

	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			set.Reindex(chain)
			_ = set.Snapshot()
			_, _ = set.FindSpendableOutputs([]byte{0x03}, 1)
			_ = set.FindUTXOs([]byte{0x03})
		}()
	}
	wait.Wait()
}

func TestUTXOSetSourceHasNoParallelMutableAuthorityV1(t *testing.T) {
	source, err := os.ReadFile("utxo.go")
	if err != nil {
		t.Fatalf("read utxo.go: %v", err)
	}

	text := string(source)

	required := []string{
		"sync.RWMutex",
		"canonical *blockchain.Blockchain",
		"canonical.UpdateUTXOSetSynchronized()",
		"canonical.StateSnapshot()",
		"canonical.FindSpendableOutputsSnapshot(",
		"canonical.FindUTXOsSnapshot(",
	}
	for _, marker := range required {
		if !strings.Contains(text, marker) {
			t.Fatalf("canonical adapter marker missing: %s", marker)
		}
	}

	forbidden := []string{
		"UTXOs map[string]",
		"set.UTXOs",
		"make(map[string][]blockchain.TransactionOutput)",
		"AllBlocksSnapshot",
		"encoding/hex",
		"spent :=",
	}
	for _, marker := range forbidden {
		if strings.Contains(text, marker) {
			t.Fatalf("parallel UTXO authority marker remains: %s", marker)
		}
	}
}
