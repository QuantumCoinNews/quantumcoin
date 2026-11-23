package blockchain

import (
	"bytes"
	"testing"

	"quantumcoin/wallet"
)

func mineReplacementBranchBlock(t *testing.T, chain *Blockchain) {
	t.Helper()
	_, err := chain.MineBlock(wallet.NewWallet().GetAddress(), testExpectedNextDifficultyV1(t, chain))
	if err != nil {
		t.Fatal(err)
	}
}

func TestReplacementReplaysCommonSpentHistoryInCandidateState(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "4")
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	if err := chain.AddTransaction(spend); err != nil {
		t.Fatal(err)
	}
	mineReplacementBranchBlock(t, chain)
	branch := DeserializeBlockchain(SerializeBlockchain(chain))
	branch.coinbaseMaturity = chain.coinbaseMaturity
	mineReplacementBranchBlock(t, branch)
	incoming := branch.BlocksSnapshot()
	if err := chain.ValidateReplacementChainStrictV1(incoming); err != nil {
		t.Fatalf("candidate replay failed: %v", err)
	}
	if err := chain.ReplaceChain(incoming); err != nil {
		t.Fatalf("valid extension rejected after replay: %v", err)
	}
	if !bytes.Equal(chain.GetLastBlock().Hash, branch.GetLastBlock().Hash) {
		t.Fatal("candidate tip was not installed")
	}
}

func TestReplacementReplaysConflictingForkInCandidateState(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "4")
	chain, sender, _, spend := newFeeConsensusTestStateV1(t)
	branch := DeserializeBlockchain(SerializeBlockchain(chain))
	branch.coinbaseMaturity = chain.coinbaseMaturity
	if err := chain.AddTransaction(spend); err != nil {
		t.Fatal(err)
	}
	mineReplacementBranchBlock(t, chain)
	conflict := cloneTransactionV1(spend)
	conflict.Outputs[0].PubKeyHash = wallet.HashPubKey(wallet.NewWallet().PublicKey)
	conflict.ID = conflict.Hash()
	if err := conflict.Sign(sender.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err := branch.AddTransaction(conflict); err != nil {
		t.Fatal(err)
	}
	mineReplacementBranchBlock(t, branch)
	mineReplacementBranchBlock(t, branch)
	if err := chain.ReplaceChain(branch.BlocksSnapshot()); err != nil {
		t.Fatalf("valid fork rejected using old branch UTXO: %v", err)
	}
	if !bytes.Equal(chain.GetLastBlock().Hash, branch.GetLastBlock().Hash) {
		t.Fatal("fork tip was not installed")
	}
}
