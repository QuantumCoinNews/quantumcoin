package blockchain

import (
	"fmt"
	"path/filepath"
	"quantumcoin/wallet"
	"testing"
	"time"
)

func TestMicroFeesSurviveSignedAdmissionMiningAndTrustedRestart(t *testing.T) {
	for _, fee := range []int{1, 10} {
		t.Run(fmt.Sprintf("%d_atoms", fee), func(t *testing.T) {
			chain := atomRewardTestChainV1(t, 1)
			owner, receiver, miner := wallet.NewWallet(), wallet.NewWallet(), wallet.NewWallet()
			if err := chain.BindRuntimeNetworkIdentityV1("local-release"); err != nil {
				t.Fatal(err)
			}
			difficulty, err := chain.ExpectedNextDifficultyV1()
			if err != nil {
				t.Fatal(err)
			}
			funding, err := chain.MineBlock(owner.GetAddress(), difficulty)
			if err != nil {
				t.Fatal(err)
			}
			input := funding.Transactions[0]
			amount := input.Outputs[0].Amount - fee
			tx := &Transaction{Inputs: []TransactionInput{{TxID: append([]byte(nil), input.ID...), OutIndex: 0}}, Outputs: []TransactionOutput{{Amount: amount, PubKeyHash: wallet.HashPubKey(receiver.PublicKey)}}, Timestamp: time.Now().UTC(), Sender: owner.GetAddress(), Amount: float64(amount)}
			tx.ID = tx.Hash()
			if err := tx.Sign(owner.PrivateKey); err != nil {
				t.Fatal(err)
			}
			if err := chain.AddTransaction(tx); err != nil {
				t.Fatal(err)
			}
			difficulty, err = chain.ExpectedNextDifficultyV1()
			if err != nil {
				t.Fatal(err)
			}
			block, err := chain.MineBlock(miner.GetAddress(), difficulty)
			if err != nil {
				t.Fatal(err)
			}
			if got := block.Transactions[0].Outputs[0].Amount; got != 50_000_000+fee {
				t.Fatalf("coinbase=%d fee=%d", got, fee)
			}
			file := filepath.Join(t.TempDir(), "chain_data.dat")
			if err := chain.SaveToFile(file); err != nil {
				t.Fatal(err)
			}
			loaded, err := LoadBlockchainFromFileWithLocalContextV1(file, chain)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.TotalMinted() != chain.TotalMinted() || loaded.Blocks[2].Transactions[0].Outputs[0].Amount != 50_000_000+fee {
				t.Fatal("restart changed atom fee or issuance")
			}
		})
	}
}
func TestConfirmedPermanentBurnDoesNotCreateReplacementMining(t *testing.T) {
	chain := atomRewardTestChainV1(t, 1)
	owner, miner := wallet.NewWallet(), wallet.NewWallet()
	difficulty, err := chain.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chain.MineBlock(owner.GetAddress(), difficulty); err != nil {
		t.Fatal(err)
	}
	before := chain.TotalMinted()
	supply := chain.TotalSupply
	burn, err := NewCanonicalBurnTransactionFromQCAmountV1(owner.GetAddress(), "0.000001", chain)
	if err != nil {
		t.Fatal(err)
	}
	if err := burn.Sign(owner.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err := chain.AddTransaction(burn); err != nil {
		t.Fatal(err)
	}
	difficulty, err = chain.ExpectedNextDifficultyV1()
	if err != nil {
		t.Fatal(err)
	}
	block, err := chain.MineBlock(miner.GetAddress(), difficulty)
	if err != nil {
		t.Fatal(err)
	}
	if block.Transactions[0].Outputs[0].Amount != 50_000_000 || chain.TotalMinted() != before+50 || chain.TotalSupply != supply {
		t.Fatal("burn caused replacement emission")
	}
	circulating, err := chain.CirculatingSupplyAtomsCheckedV1()
	if err != nil {
		t.Fatal(err)
	}
	if circulating != int64(chain.TotalMinted())*1_000_000-1 {
		t.Fatal("burn did not permanently reduce circulation")
	}
}
