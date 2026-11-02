package miner

import (
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/wallet"
)

func TestSumFeesAtomsUsesBlockchainVerifiedFeesV1(t *testing.T) {
	sender := wallet.NewWallet()
	receiver := wallet.NewWallet()

	funding := &blockchain.Transaction{
		Inputs: []blockchain.TransactionInput{},
		Outputs: []blockchain.TransactionOutput{
			{
				Amount:     5,
				PubKeyHash: wallet.HashPubKey(sender.PublicKey),
			},
		},
		Timestamp: time.Unix(blockchain.CanonicalGenesisUnixV1, 0).UTC(),
		Sender:    "COINBASE",
		Amount:    5,
	}
	funding.ID = funding.Hash()

	genesis := &blockchain.Block{
		Index:        0,
		Timestamp:    blockchain.CanonicalGenesisUnixV1,
		Transactions: []*blockchain.Transaction{funding},
		Hash:         []byte("miner-fee-test-genesis"),
		Difficulty:   1,
		Metadata:     map[string]string{},
	}

	chain := &blockchain.Blockchain{
		Blocks:      []*blockchain.Block{genesis},
		UTXO:        map[string][]blockchain.TransactionOutput{},
		TotalSupply: 1_000,
	}
	chain.UpdateUTXOSet()

	spend := &blockchain.Transaction{
		Inputs: []blockchain.TransactionInput{
			{TxID: append([]byte(nil), funding.ID...), OutIndex: 0},
		},
		Outputs: []blockchain.TransactionOutput{
			{
				Amount:     4,
				PubKeyHash: wallet.HashPubKey(receiver.PublicKey),
			},
		},
		Timestamp: time.Unix(blockchain.CanonicalGenesisUnixV1+30, 0).UTC(),
		Sender:    sender.GetAddress(),
		Amount:    4,
	}
	spend.ID = spend.Hash()
	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("sign spend: %v", err)
	}

	fees, err := SumFeesAtoms(chain, []*blockchain.Transaction{spend})
	if err != nil {
		t.Fatalf("sum verified fees: %v", err)
	}
	if fees != 1 {
		t.Fatalf("fees: got %d want 1", fees)
	}
}
