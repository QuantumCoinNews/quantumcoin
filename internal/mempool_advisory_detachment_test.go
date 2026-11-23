package internal

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"quantumcoin/ai"
	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

func mempoolAcceptanceFixture(t *testing.T) (*blockchain.Blockchain, *blockchain.Transaction) {
	t.Helper()
	owner, recipient := wallet.NewWallet(), wallet.NewWallet()
	funding := &blockchain.Transaction{Sender: "COINBASE", Amount: 1000, Timestamp: blockchain.CanonicalGenesisTimeV1(),
		Outputs: []blockchain.TransactionOutput{{Amount: 1000, PubKeyHash: wallet.HashPubKey(owner.PublicKey)}}}
	funding.ID = funding.Hash()
	hash := sha256.Sum256([]byte("test-only-funded-utxo"))
	chain := &blockchain.Blockchain{Blocks: []*blockchain.Block{{Index: 0, Hash: hash[:], Difficulty: 1,
		Timestamp: blockchain.CanonicalGenesisUnixV1, Transactions: []*blockchain.Transaction{funding}}}}
	chain.SetCoinbaseMaturity(0)
	chain.UpdateUTXOSetSynchronized()
	tx, err := blockchain.NewTransaction(owner.GetAddress(), recipient.GetAddress(), 600, chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	return chain, tx
}

func TestCanonicalMempoolAcceptsLargeTransferWithAIOnAndOff(t *testing.T) {
	t.Setenv("QC_NODE_DIR", t.TempDir())
	chain, tx := mempoolAcceptanceFixture(t)
	other := blockchain.DeserializeBlockchain(blockchain.SerializeBlockchain(chain))
	previous := ai.Enabled()
	t.Cleanup(func() { ai.SetEnabled(previous) })
	ai.SetEnabled(false)
	if !NewCanonicalMempool(chain).Add(tx) {
		t.Fatal("valid 600 QC transaction rejected with AI off")
	}
	ai.SetEnabled(true)
	if !NewCanonicalMempool(other).Add(tx) {
		t.Fatal("AI changed acceptance of the same 600 QC transaction")
	}
}

func TestCanonicalMempoolDetachesAcceptedTransactionAndReaders(t *testing.T) {
	t.Setenv("QC_NODE_DIR", t.TempDir())
	chain, tx := mempoolAcceptanceFixture(t)
	pool := NewCanonicalMempool(chain)
	if !pool.Add(tx) {
		t.Fatal("valid transaction rejected")
	}
	id := append([]byte(nil), tx.ID...)
	tx.ID[0] ^= 1
	tx.Inputs[0].Signature[0] ^= 1
	tx.Inputs[0].TxID[0] ^= 1
	tx.Outputs[0].Amount++
	tx.Outputs[0].PubKeyHash[0] ^= 1
	read := pool.GetTransactions()
	if len(read) != 1 || !bytes.Equal(read[0].ID, id) || !read[0].Verify() {
		t.Fatal("caller mutation changed the accepted transaction")
	}
	read[0].ID[0] ^= 1
	read[0].Inputs[0].Signature[0] ^= 1
	read[0].Outputs[0].Amount++
	again := pool.GetTransactions()
	if len(again) != 1 || !bytes.Equal(again[0].ID, id) || !again[0].Verify() || !pool.Has(id) || pool.Len() != 1 {
		t.Fatal("reader mutation changed canonical content or indexes")
	}
}
