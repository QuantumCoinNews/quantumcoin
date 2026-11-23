package blockchain

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"quantumcoin/wallet"
)

func TestPostEmissionPartialFeesEmptyBlocksAndTrustedReplay(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	owner, receiver, miner := wallet.NewWallet(), wallet.NewWallet(), wallet.NewWallet()
	bc := NewBlockchain(50, 0)
	// Small deterministic cap fixture, not a change to the canonical QC cap.
	bc.TotalSupply = 57
	bc.SetCoinbaseMaturity(0)
	funding := bc.Blocks[0].Transactions[0]
	funding.Outputs = []TransactionOutput{{Amount: 50, PubKeyHash: wallet.HashPubKey(owner.PublicKey)}}
	funding.Amount = 50
	funding.ID = funding.Hash()
	genesis, err := NewCanonicalGenesisBlockV1([]*Transaction{funding})
	if err != nil {
		t.Fatal(err)
	}
	bc.Blocks = []*Block{genesis}
	bc.UpdateUTXOSet()
	spend := func(prev *Transaction, w *wallet.Wallet, amount int) *Transaction {
		tx := &Transaction{Inputs: []TransactionInput{{TxID: append([]byte(nil), prev.ID...), OutIndex: 0}}, Outputs: []TransactionOutput{{Amount: amount, PubKeyHash: wallet.HashPubKey(receiver.PublicKey)}}, Timestamp: time.Now().UTC(), Sender: w.GetAddress(), Amount: float64(amount)}
		tx.ID = tx.Hash()
		if err := tx.Sign(w.PrivateKey); err != nil {
			t.Fatal(err)
		}
		if err := bc.AddTransaction(tx); err != nil {
			t.Fatal(err)
		}
		return tx
	}
	firstSpend := spend(funding, owner, 49)
	first, err := bc.MineBlock(miner.GetAddress(), 8)
	if err != nil {
		t.Fatal(err)
	}
	if first.Transactions[0].Outputs[0].Amount != 8 || bc.TotalMinted() != 57 {
		t.Fatal("final partial subsidy/fee composition wrong")
	}
	spend(firstSpend, receiver, 48)
	second, err := bc.MineBlock(miner.GetAddress(), 8)
	if err != nil {
		t.Fatalf("fees-only mining: %v", err)
	}
	if second.Transactions[0].Outputs[0].Amount != 1 || !bytes.Equal(second.Transactions[0].Outputs[0].PubKeyHash, wallet.HashPubKey(miner.PublicKey)) || bc.TotalMinted() != 57 {
		t.Fatal("fee payment minted supply or paid wrong address")
	}
	template, _, err := bc.PrepareExternalMiningTemplate(miner.GetAddress(), 8)
	if err != nil {
		t.Fatalf("empty external work: %v", err)
	}
	if template.Transactions[0].Outputs[0].Amount != 0 {
		t.Fatal("empty external work minted reward")
	}
	template.Transactions[0].Outputs[0].Amount = 1
	if _, err := bc.ValidateBlockRewardWithFeesV1(template); err == nil {
		t.Fatal("post-emission overpayment accepted")
	}
	template.Transactions[0].Outputs[0].Amount = 0
	solveExternalTemplate(t, bc, template)
	if err := bc.CommitValidatedBlockAndPersist(template, filepath.Join(t.TempDir(), "external.dat")); err != nil {
		t.Fatalf("empty external commit: %v", err)
	}
	for i := 0; i < 2; i++ {
		block, err := bc.MineBlock(miner.GetAddress(), 8)
		if err != nil {
			t.Fatalf("empty mining: %v", err)
		}
		if block.Transactions[0].Outputs[0].Amount != 0 || bc.TotalMinted() != 57 {
			t.Fatal("empty block created issuance")
		}
	}
	path := filepath.Join(t.TempDir(), "chain.dat")
	if err := bc.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(path, bc)
	if err != nil {
		t.Fatalf("trusted replay: %v", err)
	}
	if loaded.TotalMinted() != 57 || len(loaded.Blocks) != len(bc.Blocks) {
		t.Fatal("restart changed supply or height")
	}
}
