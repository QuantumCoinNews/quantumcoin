package blockchain

import (
	"strings"
	"testing"
)

func cloneGenesisTransactionForIntegrityV1(
	transaction *Transaction,
) *Transaction {
	if transaction == nil {
		return nil
	}

	cloned := *transaction
	cloned.ID = append([]byte(nil), transaction.ID...)
	cloned.Inputs = append(
		[]TransactionInput(nil),
		transaction.Inputs...,
	)
	cloned.Outputs = append(
		[]TransactionOutput(nil),
		transaction.Outputs...,
	)

	for index := range cloned.Outputs {
		cloned.Outputs[index].PubKeyHash = append(
			[]byte(nil),
			transaction.Outputs[index].PubKeyHash...,
		)
	}

	return &cloned
}

func cloneGenesisBlockForIntegrityV1(
	block *Block,
) *Block {
	if block == nil {
		return nil
	}

	cloned := *block
	cloned.PrevHash = append([]byte(nil), block.PrevHash...)
	cloned.Hash = append([]byte(nil), block.Hash...)

	cloned.Transactions = make(
		[]*Transaction,
		len(block.Transactions),
	)

	for index, transaction := range block.Transactions {
		cloned.Transactions[index] =
			cloneGenesisTransactionForIntegrityV1(transaction)
	}

	if block.Metadata != nil {
		cloned.Metadata = make(
			map[string]string,
			len(block.Metadata),
		)

		for key, value := range block.Metadata {
			cloned.Metadata[key] = value
		}
	}

	return &cloned
}

func TestValidateReplacementChainStrictV1RejectsTamperedGenesisContent(
	t *testing.T,
) {
	chain := NewBlockchain(1, 0)

	tamperedGenesis := cloneGenesisBlockForIntegrityV1(
		chain.Blocks[0],
	)

	if tamperedGenesis == nil ||
		len(tamperedGenesis.Transactions) == 0 ||
		tamperedGenesis.Transactions[0] == nil ||
		len(tamperedGenesis.Transactions[0].Outputs) == 0 {
		t.Fatal("test genesis is incomplete")
	}

	genesisTransaction := tamperedGenesis.Transactions[0]
	genesisTransaction.Outputs[0].Amount++
	genesisTransaction.Amount =
		float64(genesisTransaction.Outputs[0].Amount)
	genesisTransaction.ID = genesisTransaction.Hash()

	reward := GetCurrentReward()
	if reward <= 0 {
		reward = 1
	}

	next, err := NewBlock(
		1,
		[]*Transaction{
			testReplacementCoinbaseTxV1(reward),
		},
		tamperedGenesis.Hash,
		"replacement-miner",
		1,
	)
	if err != nil {
		t.Fatalf("mine replacement block: %v", err)
	}

	incoming := []*Block{
		tamperedGenesis,
		next,
	}

	err = chain.ValidateReplacementChainStrictV1(incoming)
	if err == nil {
		t.Fatal(
			"replacement validation accepted tampered genesis content with copied hash",
		)
	}

	if !strings.Contains(
		strings.ToLower(err.Error()),
		"genesis",
	) && !strings.Contains(
		strings.ToLower(err.Error()),
		"hash",
	) {
		t.Fatalf(
			"expected genesis/hash integrity error, got: %v",
			err,
		)
	}
}
