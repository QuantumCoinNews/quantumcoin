package blockchain

import (
	"path/filepath"
	"quantumcoin/config"
	"quantumcoin/wallet"
	"strings"
	"testing"
	"time"
)

func TestFounderBindingPeerReorgAndTrustedReplay(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	bc, owner, funding := founderBoundaryFixtureV1(t)
	path := filepath.Join(t.TempDir(), "base.dat")
	if err := bc.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	fork, err := LoadBlockchainFromFileWithLocalContextV1(path, bc)
	if err != nil {
		t.Fatal(err)
	}
	key := wallet.HashPubKey(owner.PublicKey)
	first := &Transaction{Inputs: []TransactionInput{{TxID: funding.ID, OutIndex: 0}}, Outputs: []TransactionOutput{{Amount: FounderLockedAllocationV1, PubKeyHash: key}}, Timestamp: time.Unix(bc.Blocks[0].Timestamp+30, 0), Sender: owner.GetAddress(), Amount: float64(FounderLockedAllocationV1)}
	first.ID = first.Hash()
	if err := first.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	second := &Transaction{Inputs: []TransactionInput{{TxID: first.ID, OutIndex: 0}}, Outputs: []TransactionOutput{{Amount: FounderLockedAllocationV1 - 1, PubKeyHash: key}, {Amount: 1, PubKeyHash: wallet.HashPubKey(wallet.NewWallet().PublicKey)}}, Timestamp: first.Timestamp, Sender: owner.GetAddress(), Amount: float64(FounderLockedAllocationV1)}
	second.ID = second.Hash()
	if err := second.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	miner := wallet.NewWallet().GetAddress()
	makeCandidate := func(at int64) *Block {
		cb, err := newCoinbaseTxWithRewardV1(miner, 50, time.Unix(at, 0))
		if err != nil {
			t.Fatal(err)
		}
		b, err := newBlockAtTimeV1(1, []*Transaction{cb, first, second}, bc.Blocks[0].Hash, miner, testExpectedNextDifficultyV1(t, bc), at)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if err := bc.AddBlockFromPeerSynchronizedV2(makeCandidate(first.Timestamp.Unix())); err == nil || !strings.Contains(err.Error(), "founder") {
		t.Fatalf("peer did not reject early release: %v", err)
	}
	if err := bc.AddBlockFromPeerSynchronizedV2(makeCandidate(CanonicalGenesisTimeV1().AddDate(1, 1, 0).Unix())); err != nil {
		t.Fatalf("peer rejected valid unlocked sequence: %v", err)
	}
	released, err := bc.founderReleasedConfirmedAtHeightV1(key, 2)
	if err != nil || released != 1 {
		t.Fatal("confirmed founder release mismatch", released, err)
	}
	if err := bc.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	previous := config.Current()
	t.Cleanup(func() { config.Set(previous) })
	cfg := config.Default()
	cfg.PremineLockedAddress = wallet.NewWallet().GetAddress()
	cfg.PremineLiquidAddress = wallet.NewWallet().GetAddress()
	cfg.GenesisUnix = 1
	config.Set(cfg)
	t.Setenv("QC_MAIN_ADDRESS", wallet.NewWallet().GetAddress())
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(path, bc)
	if err != nil {
		t.Fatalf("trusted replay after config drift: %v", err)
	}
	if released, err := loaded.founderReleasedConfirmedAtHeightV1(key, 2); err != nil || released != 1 {
		t.Fatal("restart release mismatch", released, err)
	}
	for i := 0; i < 2; i++ {
		if _, err := fork.MineBlock(miner, testExpectedNextDifficultyV1(t, fork)); err != nil {
			t.Fatal(err)
		}
	}
	if err := loaded.ReplaceChainFromPeerStrictV1(fork.BlocksSnapshot()); err != nil {
		t.Fatalf("valid founder reorg: %v", err)
	}
	if released, err := loaded.founderReleasedConfirmedAtHeightV1(key, 3); err != nil || released != 0 {
		t.Fatal("orphan founder release remained counted", released, err)
	}
	if _, err := loaded.ValidateTransactionUTXOV1(second, false); err == nil {
		t.Fatal("orphan intermediate output was accepted as spendable")
	}
	earlySpend := *second
	earlySpend.Inputs = []TransactionInput{{TxID: funding.ID, OutIndex: 0}}
	earlySpend.ID = earlySpend.Hash()
	if err := earlySpend.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err := loaded.ValidateFounderVestingSpendV1(&earlySpend, bc.Blocks[0].Timestamp+30); err == nil {
		t.Fatal("config/env/reorg disabled genesis cliff")
	}
}

func founderBoundaryFixtureV1(t *testing.T) (*Blockchain, *wallet.Wallet, *Transaction) {
	t.Helper()
	owner := wallet.NewWallet()
	bc := NewBlockchain(50, 0)
	funding := &Transaction{Outputs: []TransactionOutput{{Amount: FounderLockedAllocationV1, PubKeyHash: wallet.HashPubKey(owner.PublicKey)}}, Timestamp: CanonicalGenesisTimeV1(), Sender: "COINBASE", Amount: float64(FounderLockedAllocationV1)}
	funding.ID = funding.Hash()
	genesis, err := NewCanonicalGenesisBlockV1([]*Transaction{funding})
	if err != nil {
		t.Fatal(err)
	}
	bc.Blocks = []*Block{genesis}
	bc.UpdateUTXOSet()
	bc.SetCoinbaseMaturity(0)
	return bc, owner, funding
}

func TestFounderBindingAmbiguousGenesisFailsClosed(t *testing.T) {
	bc, owner, funding := founderBoundaryFixtureV1(t)
	funding.Outputs = append(funding.Outputs, TransactionOutput{Amount: FounderLockedAllocationV1, PubKeyHash: wallet.HashPubKey(wallet.NewWallet().PublicKey)})
	spend := &Transaction{Inputs: []TransactionInput{{TxID: funding.ID, OutIndex: 0}}, Outputs: []TransactionOutput{{Amount: 1, PubKeyHash: wallet.HashPubKey(owner.PublicKey)}}}
	if err := bc.ValidateFounderVestingSpendV1(spend, bc.Blocks[0].Timestamp+30); err == nil {
		t.Fatal("ambiguous genesis disabled founder vesting")
	}
	if err := bc.ValidateFounderVestingBlockV1(&Block{Index: 1, Timestamp: bc.Blocks[0].Timestamp + 30, Transactions: []*Transaction{spend}}); err == nil {
		t.Fatal("ambiguous genesis disabled block vesting")
	}
}

func TestFounderBindingIntraBlockLockedChangeCannotBypassCliff(t *testing.T) {
	bc, owner, funding := founderBoundaryFixtureV1(t)
	key := wallet.HashPubKey(owner.PublicKey)
	first := &Transaction{Inputs: []TransactionInput{{TxID: funding.ID, OutIndex: 0}}, Outputs: []TransactionOutput{{Amount: FounderLockedAllocationV1, PubKeyHash: key}}, Timestamp: time.Unix(bc.Blocks[0].Timestamp+30, 0), Sender: owner.GetAddress(), Amount: float64(FounderLockedAllocationV1)}
	first.ID = first.Hash()
	if err := first.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	second := &Transaction{Inputs: []TransactionInput{{TxID: first.ID, OutIndex: 0}}, Outputs: []TransactionOutput{{Amount: FounderLockedAllocationV1 - 1, PubKeyHash: key}, {Amount: 1, PubKeyHash: wallet.HashPubKey(wallet.NewWallet().PublicKey)}}, Timestamp: first.Timestamp, Sender: owner.GetAddress(), Amount: float64(FounderLockedAllocationV1)}
	second.ID = second.Hash()
	if err := second.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if _, err := bc.ValidateTransactionsUTXOAtHeightV1([]*Transaction{first, second}, 1); err != nil {
		t.Fatalf("signed UTXO sequence control: %v", err)
	}
	candidate := &Block{Index: 1, Timestamp: first.Timestamp.Unix(), Transactions: []*Transaction{first, second}}
	if err := bc.ValidateFounderVestingBlockV1(candidate); err == nil {
		t.Fatal("intra-block locked change bypassed founder cliff")
	}
	candidate.Timestamp = CanonicalGenesisTimeV1().AddDate(1, 1, 0).Unix()
	if err := bc.ValidateFounderVestingBlockV1(candidate); err != nil {
		t.Fatalf("unlocked valid release rejected: %v", err)
	}
}
