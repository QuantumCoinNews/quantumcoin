package blockchain

import (
	"testing"
	"time"

	"quantumcoin/wallet"
)

func advisorySignedTx(t *testing.T, owner *wallet.Wallet, input byte, amount int) *Transaction {
	t.Helper()
	tx := &Transaction{
		Sender: owner.GetAddress(), Amount: 99_999, Timestamp: time.Unix(10, 0),
		Inputs: []TransactionInput{{TxID: make([]byte, 32), OutIndex: 0}},
		Outputs: []TransactionOutput{
			{Amount: amount, PubKeyHash: wallet.HashPubKey(wallet.NewWallet().PublicKey)},
			{Amount: 19, PubKeyHash: wallet.HashPubKey(owner.PublicKey)},
		},
	}
	tx.Inputs[0].TxID[0] = input
	tx.ID = tx.Hash()
	if err := tx.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestAIAnalysisSampleUnitsDedupAndDetachment(t *testing.T) {
	chain := atomRewardTestChainV1(t, 1)
	owner := wallet.NewWallet()
	legacy := advisorySignedTx(t, owner, 1, 500)
	atom := advisorySignedTx(t, owner, 2, 500_000_000)
	observed := time.Unix(200, 0).UTC()
	chain.Blocks[0].Transactions = append(chain.Blocks[0].Transactions, legacy)
	chain.pendingTxs = []*Transaction{atom}
	chain.pendingObservedAt = map[string]time.Time{transactionLocationIndexKey(atom.ID): observed}
	lite, truncated := chain.AIAnalysisSample("")
	if truncated || len(lite) != 2 {
		t.Fatal("sample", len(lite), truncated)
	}
	for _, tx := range lite {
		if tx.Amount != 500 || tx.Timestamp.Equal(legacy.Timestamp) || tx.WalletAddress != owner.GetAddress() {
			t.Fatal("metadata trusted instead of observed canonical fields", tx)
		}
	}
	if !lite[0].Timestamp.Equal(observed) || lite[1].Timestamp.Unix() != chain.Blocks[0].Timestamp {
		t.Fatal("missing canonical observation timestamps")
	}
	lite[0].Amount = 1
	if atom.Outputs[0].Amount != 500_000_000 {
		t.Fatal("aliases canonical transaction")
	}
	chain.Blocks = append(chain.Blocks, &Block{Index: 1, Timestamp: 300, Transactions: []*Transaction{atom}})
	lite, _ = chain.AIAnalysisSample("")
	if len(lite) != 2 {
		t.Fatal("duplicate counted across pending/confirmed")
	}
	if lite, _ = chain.AIAnalysisSample("someone-else"); len(lite) != 0 {
		t.Fatal("address filter ignored")
	}
}

func TestAIAnalysisRejectsUnsignedTamperedAndUnobservedEntries(t *testing.T) {
	chain := NewBlockchain(1, 0)
	owner := wallet.NewWallet()
	tampered := advisorySignedTx(t, owner, 1, 500)
	tampered.Outputs[0].Amount++
	unsigned := advisorySignedTx(t, owner, 2, 500)
	unsigned.Inputs[0].Signature = nil
	unobserved := advisorySignedTx(t, owner, 3, 500)
	chain.pendingTxs = []*Transaction{tampered, unsigned, unobserved}
	chain.pendingObservedAt = map[string]time.Time{
		transactionLocationIndexKey(tampered.ID): time.Now(),
		transactionLocationIndexKey(unsigned.ID): time.Now(),
	}
	if lite, truncated := chain.AIAnalysisSample(""); len(lite) != 0 || !truncated {
		t.Fatal("unverified sample contributed to analysis")
	}
	for i := 0; i <= AnalysisMaxTransactions; i++ {
		chain.pendingTxs = append(chain.pendingTxs, nil)
	}
	if lite, truncated := chain.AIAnalysisSample(""); len(lite) != 0 || !truncated {
		t.Fatal("malformed entries bypassed analysis budget")
	}
}
