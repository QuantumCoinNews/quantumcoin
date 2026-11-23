package blockchain

import (
	"testing"
	"time"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func authorizationCandidateBlock(t *testing.T, chain *Blockchain, txs ...*Transaction) *Block {
	t.Helper()
	tip := chain.BlockByIndexSnapshot(chain.BestHeightSnapshot())
	timestamp := time.Unix(tip.Timestamp+30, 0).UTC()
	miner := wallet.NewWallet().GetAddress()
	coinbase, err := newCoinbaseTxWithRewardV1(miner, 50, timestamp)
	if err != nil {
		t.Fatal(err)
	}
	if err = chain.assignTransactionIDLockedV2(coinbase); err != nil {
		t.Fatal(err)
	}
	block, err := newBlockAtTimeV1(tip.Index+1, append([]*Transaction{coinbase}, txs...), tip.Hash, miner, 8, timestamp.Unix())
	if err != nil {
		t.Fatal(err)
	}
	return block
}

func rejectAuthorizationEveryAdmission(t *testing.T, chain *Blockchain, tx *Transaction) {
	t.Helper()
	for _, path := range []struct {
		name   string
		accept func(*Transaction) error
	}{
		{"mempool", chain.AddTransaction}, {"peer_transaction", chain.AddTransactionFromPeerStrictV1},
	} {
		t.Run(path.name, func(t *testing.T) {
			if err := path.accept(cloneTransactionV1(tx)); err == nil {
				t.Fatal("unauthorized transaction accepted")
			}
		})
	}
	if err := chain.ValidatePeerBlockStrictV2(authorizationCandidateBlock(t, chain)); err != nil {
		t.Fatal("positive control block invalid", err)
	}
	block := authorizationCandidateBlock(t, chain, cloneTransactionV1(tx))
	// A valid PoW must not make the transaction authorized.
	for _, path := range []struct {
		name   string
		accept func(*Block) error
	}{
		{"validate_block", chain.ValidatePeerBlockStrictV2}, {"accept_block", chain.AddBlockFromPeerStrictV2},
	} {
		t.Run(path.name, func(t *testing.T) {
			if err := path.accept(cloneBlockV1(block)); err == nil {
				t.Fatal("unauthorized spend accepted inside PoW block")
			}
		})
	}
	t.Run("replacement_replay", func(t *testing.T) {
		if err := chain.ValidateReplacementChainStrictV1(append(chain.BlocksSnapshot(), block)); err == nil {
			t.Fatal("reorg bypassed authorization")
		}
	})
	if chain.BestHeightSnapshot() != 1 || len(chain.PendingTransactionsSnapshot()) != 0 {
		t.Fatal("failed admission changed chain or mempool")
	}
}

func TestFreshAuthorizationBindsSignedFieldsAcrossAdmissionPaths(t *testing.T) {
	for _, alter := range []struct {
		name   string
		mutate func(*Transaction)
	}{
		{"missing_signature", func(tx *Transaction) { tx.Inputs[0].Signature = nil }},
		{"invalid_signature", func(tx *Transaction) { tx.Inputs[0].Signature[0] ^= 1 }},
		{"missing_public_key", func(tx *Transaction) { tx.Inputs[0].PubKey = nil }},
		{"recipient", func(tx *Transaction) { tx.Outputs[0].PubKeyHash = wallet.HashPubKey(wallet.NewWallet().PublicKey) }},
		{"amount_preserving_output_total", func(tx *Transaction) { tx.Outputs[0].Amount++; tx.Outputs[1].Amount-- }},
		{"convenience_amount", func(tx *Transaction) { tx.Amount++ }},
		{"input_id", func(tx *Transaction) { tx.Inputs[0].TxID[0] ^= 1 }},
		{"input_index", func(tx *Transaction) { tx.Inputs[0].OutIndex++ }},
	} {
		t.Run(alter.name, func(t *testing.T) {
			chain := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
			owner, recipient := wallet.NewWallet(), wallet.NewWallet()
			addSignatureFundingBlock(t, chain, owner)
			tx, err := NewTransaction(owner.GetAddress(), recipient.GetAddress(), 5, chain)
			if err != nil {
				t.Fatal(err)
			}
			if err = chain.SignTransaction(tx, owner.PrivateKey); err != nil {
				t.Fatal(err)
			}
			if !chain.VerifyTransaction(tx) {
				t.Fatal("positive control signature failed")
			}
			alter.mutate(tx)
			// Recompute the content ID so rejection cannot be credited only to stale ID.
			tx.ID, err = CanonicalTransactionIDV2(tx)
			if err != nil {
				t.Fatal(err)
			}
			if chain.VerifyTransaction(tx) {
				t.Fatal("signature did not bind the changed field")
			}
			rejectAuthorizationEveryAdmission(t, chain, tx)
		})
	}
}

func TestFreshAuthorizationValidAttackerSignatureCannotSpendVictimUTXO(t *testing.T) {
	chain := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
	victim, attacker := wallet.NewWallet(), wallet.NewWallet()
	addSignatureFundingBlock(t, chain, victim)
	theft, err := NewTransaction(victim.GetAddress(), attacker.GetAddress(), 5, chain)
	if err != nil {
		t.Fatal(err)
	}
	theft.Sender = attacker.GetAddress()
	if err = chain.SignTransaction(theft, attacker.PrivateKey); err != nil {
		t.Fatal(err)
	}
	if !chain.VerifyTransaction(theft) {
		t.Fatal("probe requires a valid attacker signature, not malformed crypto")
	}
	rejectAuthorizationEveryAdmission(t, chain, theft)
	if chain.BalanceSnapshot(attacker.GetAddress()) != 0 || chain.BalanceSnapshot(victim.GetAddress()) != 50 {
		t.Fatal("victim balance changed")
	}
}

func TestFreshAuthorizationDoubleSpendAndConflictingReorgUseCanonicalUTXO(t *testing.T) {
	chain := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
	branch := freshSignatureChain(t, config.QCNetworkProfileTestnetPublicV1)
	owner, left, right := wallet.NewWallet(), wallet.NewWallet(), wallet.NewWallet()
	funding := addSignatureFundingBlock(t, chain, owner)
	if err := branch.AddBlockFromPeerStrictV2(cloneBlockV1(funding)); err != nil {
		t.Fatal(err)
	}
	first, err := NewTransaction(owner.GetAddress(), left.GetAddress(), 5, chain)
	if err != nil {
		t.Fatal(err)
	}
	conflict, err := NewTransaction(owner.GetAddress(), right.GetAddress(), 6, chain)
	if err != nil {
		t.Fatal(err)
	}
	for _, tx := range []*Transaction{first, conflict} {
		if err = chain.SignTransaction(tx, owner.PrivateKey); err != nil {
			t.Fatal(err)
		}
	}
	if err = chain.AddTransaction(first); err != nil {
		t.Fatal(err)
	}
	if err = chain.AddTransaction(conflict); err == nil {
		t.Fatal("mempool failed to reserve spent outpoint")
	}
	if err = chain.AddTransactionFromPeerStrictV1(conflict); err == nil {
		t.Fatal("peer bypassed pending reservation")
	}
	invalid := authorizationCandidateBlock(t, chain, cloneTransactionV1(first), cloneTransactionV1(conflict))
	if err = chain.ValidatePeerBlockStrictV2(invalid); err == nil {
		t.Fatal("block double spend accepted")
	}
	if err = chain.ValidateReplacementChainStrictV1(append(chain.BlocksSnapshot(), invalid)); err == nil {
		t.Fatal("replacement double spend accepted")
	}
	if _, err = chain.MineBlock(owner.GetAddress(), 8); err != nil {
		t.Fatal(err)
	}
	if err = chain.AddTransaction(conflict); err == nil {
		t.Fatal("already-spent outpoint accepted")
	}
	if err = branch.AddTransaction(conflict); err != nil {
		t.Fatal("valid competing fork rejected", err)
	}
	for i := 0; i < 2; i++ {
		if _, err = branch.MineBlock(owner.GetAddress(), 8); err != nil {
			t.Fatal(err)
		}
	}
	if err = chain.ReplaceChainFromPeerStrictV1(branch.BlocksSnapshot()); err != nil {
		t.Fatal(err)
	}
	if chain.BalanceSnapshot(left.GetAddress()) != 0 || chain.BalanceSnapshot(right.GetAddress()) != 6 || chain.TotalMintedSnapshot() != 150 {
		t.Fatal("reorg retained both conflicting rewards/spends")
	}
	if err = chain.AddTransaction(first); err == nil {
		t.Fatal("old branch spend accepted after reorg")
	}
}

func TestFreshAuthorizationCoinbaseRequiresBlockConsensus(t *testing.T) {
	chain := freshSignatureChain(t, config.QCNetworkProfileTestnetLocalV1)
	owner := wallet.NewWallet()
	funding := addSignatureFundingBlock(t, chain, owner)
	stamp := time.Unix(funding.Timestamp+30, 0).UTC()
	coinbase, err := newCoinbaseTxWithRewardV1(owner.GetAddress(), 50, stamp)
	if err != nil {
		t.Fatal(err)
	}
	if err = chain.assignTransactionIDLockedV2(coinbase); err != nil {
		t.Fatal(err)
	}
	if err = chain.AddTransaction(coinbase); err == nil {
		t.Fatal("user mempool accepted coinbase")
	}
	if err = chain.AddTransactionFromPeerStrictV1(coinbase); err == nil {
		t.Fatal("peer transaction route accepted coinbase")
	}
	if err = chain.ValidatePeerBlockStrictV2(authorizationCandidateBlock(t, chain, coinbase)); err == nil {
		t.Fatal("multiple coinbases accepted")
	}
	forged, err := newCoinbaseTxWithRewardV1(owner.GetAddress(), 51, stamp)
	if err != nil {
		t.Fatal(err)
	}
	if err = chain.assignTransactionIDLockedV2(forged); err != nil {
		t.Fatal(err)
	}
	block, err := newBlockAtTimeV1(2, []*Transaction{forged}, funding.Hash, owner.GetAddress(), 8, stamp.Unix())
	if err != nil {
		t.Fatal(err)
	}
	if err = chain.ValidatePeerBlockStrictV2(block); err == nil {
		t.Fatal("valid PoW authorized excess subsidy")
	}
	if err = chain.ValidateReplacementChainStrictV1(append(chain.BlocksSnapshot(), block)); err == nil {
		t.Fatal("reorg authorized excess subsidy")
	}
	if err = chain.AddBlockFromPeerStrictV2(authorizationCandidateBlock(t, chain)); err != nil {
		t.Fatal("valid consensus coinbase rejected", err)
	}
	if chain.TotalMintedSnapshot() != 100 {
		t.Fatal("coinbase accounting changed")
	}
}
