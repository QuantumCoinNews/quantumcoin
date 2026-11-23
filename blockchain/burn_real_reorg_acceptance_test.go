package blockchain

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quantumcoin/consensus"
	"quantumcoin/wallet"
)

func TestBurnRealForkReplacementRestoresCanonicalSupply(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "8")
	bc := atomRewardTestChainV1(t, 1)
	if err := bc.BindRuntimeNetworkIdentityV1("local-release"); err != nil {
		t.Fatal(err)
	}
	owner, miner := wallet.NewWallet(), wallet.NewWallet()
	if _, err := bc.MineBlock(owner.GetAddress(), testExpectedNextDifficultyV1(t, bc)); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(t.TempDir(), "base.dat")
	if err := bc.SaveToFile(base); err != nil {
		t.Fatal(err)
	}
	fork, err := LoadBlockchainFromFileWithLocalContextV1(base, bc)
	if err != nil {
		t.Fatal(err)
	}
	burn, err := NewCanonicalBurnTransactionFromQCAmountV1(owner.GetAddress(), "0.000001", bc)
	if err != nil {
		t.Fatal(err)
	}
	if err := burn.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err := bc.AddTransaction(burn); err != nil {
		t.Fatal(err)
	}
	if got, err := bc.TotalBurnedAtomsCheckedV1(); err != nil || got != 0 {
		t.Fatal("pending burn changed confirmed accounting", got, err)
	}
	if _, err := bc.MineBlock(miner.GetAddress(), testExpectedNextDifficultyV1(t, bc)); err != nil {
		t.Fatal(err)
	}
	if got, err := bc.TotalBurnedAtomsCheckedV1(); err != nil || got != 1 {
		t.Fatal("confirmed burn missing", got, err)
	}
	attempt := &Transaction{Inputs: []TransactionInput{{TxID: append([]byte(nil), burn.ID...), OutIndex: 0}}, Outputs: []TransactionOutput{{Amount: 1, PubKeyHash: wallet.HashPubKey(owner.PublicKey)}}, Timestamp: time.Now().UTC(), Sender: owner.GetAddress(), Amount: 1}
	attempt.ID = attempt.Hash()
	if err := attempt.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if _, err := bc.ValidateTransactionUTXOV1(attempt, false); err == nil || !strings.Contains(err.Error(), "canonical burn output is unspendable") {
		t.Fatalf("confirmed burn spend was not rejected at burn boundary: %v", err)
	}
	if err := bc.AddTransaction(attempt); err == nil {
		t.Fatal("confirmed burn output reached mempool")
	}
	for i := 0; i < 2; i++ {
		if _, err := fork.MineBlock(miner.GetAddress(), testExpectedNextDifficultyV1(t, fork)); err != nil {
			t.Fatal(err)
		}
	}
	if err := bc.ReplaceChainFromPeerStrictV1(fork.BlocksSnapshot()); err != nil {
		t.Fatalf("real higher-work fork rejected: %v", err)
	}
	if !bytes.Equal(bc.TipSnapshot().Hash, fork.TipSnapshot().Hash) {
		t.Fatal("fork tip not adopted")
	}
	if got, err := bc.TotalBurnedAtomsCheckedV1(); err != nil || got != 0 {
		t.Fatal("orphan burn retained in canonical accounting", got, err)
	}
	circulating, err := bc.CirculatingSupplyAtomsCheckedV1()
	issued, convertErr := consensus.WholeQCToAtomsV1(int64(bc.TotalMinted()))
	if err != nil || convertErr != nil || circulating != issued || bc.TotalSupply != int(QCTotalSupplyHardCap) {
		t.Fatal("reorg changed cap or circulating supply", err, convertErr)
	}
	path := filepath.Join(t.TempDir(), "reorg.dat")
	if err := bc.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBlockchainFromFileWithLocalContextV1(path, bc)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := loaded.TotalBurnedAtomsCheckedV1(); err != nil || got != 0 {
		t.Fatal("restart restored orphan burn", got, err)
	}
	if got, err := loaded.CirculatingSupplyAtomsCheckedV1(); err != nil || got != issued {
		t.Fatal("restart circulation mismatch", got, err)
	}
}

func TestBurnMarkerCannotUnlockItselfOrAliasFactory(t *testing.T) {
	first := NewCanonicalBurnOutputV1(1)
	if wallet.ValidateAddress(consensus.CanonicalBurnSinkAddressV1) {
		t.Fatal("burn alias accepted as a wallet address")
	}
	if first.IsLockedWithKey(first.PubKeyHash) {
		t.Fatal("burn marker unlocked by itself")
	}
	first.PubKeyHash[0] ^= 0xff
	if !IsCanonicalBurnOutputV1(NewCanonicalBurnOutputV1(1)) {
		t.Fatal("caller output mutation changed canonical marker")
	}
}
