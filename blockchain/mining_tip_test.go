package blockchain

import (
	"encoding/hex"
	"testing"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestMiningReceiptFollowsCanonicalForkWithoutCreditingBothBranches(t *testing.T) {
	chain := freshTimeChain(t, config.QCNetworkProfileTestnetLocalV1)
	branch := freshTimeChain(t, config.QCNetworkProfileTestnetLocalV1)
	// Production bootstrap installs maturity; bare chain constructors do not.
	chain.SetCoinbaseMaturity(10)
	branch.SetCoinbaseMaturity(10)
	owner, other := wallet.NewWallet().GetAddress(), wallet.NewWallet().GetAddress()
	old, err := chain.MineBlock(owner, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !chain.MiningResultSnapshot(1, old.Hash, owner) {
		t.Fatal("missing original receipt")
	}
	for i := 0; i < 2; i++ {
		if _, err = branch.MineBlock(other, 8); err != nil {
			t.Fatal(err)
		}
	}
	if err = chain.ReplaceChainFromPeerStrictV1(branch.BlocksSnapshot()); err != nil {
		t.Fatal(err)
	}
	if chain.MiningResultSnapshot(1, old.Hash, owner) {
		t.Fatal("orphaned block still reported canonical")
	}
	tip := chain.MiningTipSnapshot()
	if tip.Height != 2 || tip.Hash != hex.EncodeToString(branch.BlockByIndexSnapshot(2).Hash) {
		t.Fatal("tip did not follow winning branch")
	}
	if chain.TotalMintedSnapshot() != 100 || chain.BalanceSnapshot(owner) != 0 {
		t.Fatal("reward from losing branch retained")
	}
	if chain.coinbaseMaturity != 10 {
		t.Fatal("solo mining changed maturity", chain.coinbaseMaturity)
	}
}
