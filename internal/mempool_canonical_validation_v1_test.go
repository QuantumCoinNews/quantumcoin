package internal

import (
	"os"
	"strings"
	"testing"

	"quantumcoin/blockchain"
)

func TestMempoolRequiresCanonicalBindingV1(t *testing.T) {
	mp := NewMempool()
	if mp == nil {
		t.Fatal("NewMempool returned nil")
	}

	tx := &blockchain.Transaction{
		ID: []byte("qma0040-unbound"),
		Inputs: []blockchain.TransactionInput{{
			TxID:     []byte("missing-prevout"),
			OutIndex: 0,
		}},
		Outputs: []blockchain.TransactionOutput{{
			Amount:     1,
			PubKeyHash: []byte{1},
		}},
	}

	if mp.Add(tx) {
		t.Fatal("unbound internal mempool accepted a transaction")
	}

	if mp.Len() != 0 {
		t.Fatalf("unbound mempool length=%d want=0", mp.Len())
	}
}

func TestMempoolExplicitlyRejectsCoinbaseV1(t *testing.T) {
	chain := blockchain.NewBlockchain(1, 0)
	mp := NewCanonicalMempool(chain)

	coinbase := &blockchain.Transaction{
		ID: []byte("qma0040-coinbase"),
		Outputs: []blockchain.TransactionOutput{{
			Amount:     1,
			PubKeyHash: []byte{1},
		}},
	}

	if !coinbase.IsCoinbase() {
		t.Fatal("coinbase fixture is not coinbase")
	}

	if mp.Add(coinbase) {
		t.Fatal("internal mempool accepted coinbase")
	}

	if chain.PendingTransactionCountSnapshot() != 0 {
		t.Fatal("coinbase reached canonical pending pool")
	}
}

func TestMempoolDelegatesAdmissionToCanonicalBlockchainV1(
	t *testing.T,
) {
	chain := blockchain.NewBlockchain(1, 0)
	mp := NewCanonicalMempool(chain)

	invalidSpend := &blockchain.Transaction{
		ID: []byte("qma0040-invalid-spend"),
		Inputs: []blockchain.TransactionInput{{
			TxID:      []byte("missing-prevout"),
			OutIndex:  0,
			Signature: []byte{1},
			PubKey:    []byte{4, 1},
		}},
		Outputs: []blockchain.TransactionOutput{{
			Amount:     1,
			PubKeyHash: []byte{1},
		}},
		Sender: "invalid-owner",
		Amount: 1,
	}

	if mp.Add(invalidSpend) {
		t.Fatal("canonical-invalid spend was accepted")
	}

	if chain.PendingTransactionCountSnapshot() != 0 {
		t.Fatal("rejected spend changed canonical pending state")
	}

	if mp.Len() != chain.PendingTransactionCountSnapshot() {
		t.Fatal("adapter count diverged from canonical count")
	}
}

func TestMempoolCannotSwitchCanonicalAuthorityV1(t *testing.T) {
	first := blockchain.NewBlockchain(1, 0)
	second := blockchain.NewBlockchain(1, 0)

	mp := NewMempool()

	if !mp.BindCanonicalBlockchain(first) {
		t.Fatal("initial canonical binding rejected")
	}

	if !mp.BindCanonicalBlockchain(first) {
		t.Fatal("idempotent canonical binding rejected")
	}

	if mp.BindCanonicalBlockchain(second) {
		t.Fatal("adapter switched to a different canonical Blockchain")
	}

	if mp.BindCanonicalBlockchain(nil) {
		t.Fatal("nil canonical binding accepted")
	}
}

func TestMempoolLegacyMutatorsCannotCreateSecondPoolV1(
	t *testing.T,
) {
	chain := blockchain.NewBlockchain(1, 0)
	mp := NewCanonicalMempool(chain)

	if batch := mp.PopBatch(10); batch != nil {
		t.Fatal("PopBatch returned independently owned transactions")
	}

	if mp.RemoveTx([]byte("unknown")) {
		t.Fatal("RemoveTx reported independent mutation")
	}

	mp.Clear()

	if mp.Len() != chain.PendingTransactionCountSnapshot() {
		t.Fatal("legacy mutator caused canonical/adapter divergence")
	}

	if transactions := mp.GetTransactions(); len(transactions) != 0 {
		t.Fatalf(
			"unexpected canonical pending transactions: %d",
			len(transactions),
		)
	}
}

func TestMempoolSourceUsesCanonicalAdmissionOnlyV1(t *testing.T) {
	source, err := os.ReadFile("mempool.go")
	if err != nil {
		t.Fatalf("read mempool.go: %v", err)
	}

	text := string(source)

	required := []string{
		"if tx.IsCoinbase()",
		"canonical.AddTransaction(tx)",
		"PendingTransactionCountSnapshot",
		"PendingTransactionsSnapshot",
	}

	for _, marker := range required {
		if !strings.Contains(text, marker) {
			t.Fatalf("canonical adapter marker missing: %s", marker)
		}
	}

	forbidden := []string{
		"AnalyzeTransactions",
		"append(mp.transactions",
		"transactions []*blockchain.Transaction",
		"index map[string]struct{}",
	}

	for _, marker := range forbidden {
		if strings.Contains(text, marker) {
			t.Fatalf("independent admission marker remains: %s", marker)
		}
	}
}
