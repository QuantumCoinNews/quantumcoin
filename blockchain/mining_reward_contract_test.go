package blockchain

import (
	"bytes"
	"quantumcoin/wallet"
	"testing"
)

// Exercise the real local miner with a signed one-atom fee; reward validation
// and UTXO application must agree on the single coinbase recipient.
func TestMiningCoinbasePaysSubsidyAndVerifiedFeeToMiner(t *testing.T) {
	bc := atomRewardTestChainV1(t, 1)
	spend, _ := atomRewardOneAtomFeeFixtureV1(t, bc)
	if err := bc.AddTransaction(spend); err != nil {
		t.Fatal(err)
	}
	owner := wallet.NewWallet()
	block, err := bc.MineBlock(owner.GetAddress(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(block.Transactions) != 2 {
		t.Fatalf("transactions=%d", len(block.Transactions))
	}
	coinbase := block.Transactions[0]
	if len(coinbase.Outputs) != 1 || coinbase.Outputs[0].Amount != 50_000_001 || !bytes.Equal(coinbase.Outputs[0].PubKeyHash, wallet.HashPubKey(owner.PublicKey)) {
		t.Fatalf("unexpected coinbase: %+v", coinbase)
	}
}
