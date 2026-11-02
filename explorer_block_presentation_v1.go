package main

import (
	"time"

	"quantumcoin/blockchain"
)

type explorerAmountConverterV1 interface {
	TransactionAmountQCAtHeightV1(
		amount float64,
		height int,
	) float64
}

type explorerTransactionOutputV1 struct {
	Amount     float64
	PubKeyHash []byte
}

type explorerTransactionV1 struct {
	ID        []byte
	Inputs    []blockchain.TransactionInput
	Outputs   []explorerTransactionOutputV1
	Timestamp time.Time
	Sender    string
	Amount    float64
}

type explorerBlockV1 struct {
	Index        int
	Timestamp    int64
	Transactions []*explorerTransactionV1
	PrevHash     []byte
	Hash         []byte
	Nonce        int
	Miner        string
	Difficulty   int
	Metadata     map[string]string
}

// newBlockDetailPresentationV1 converts protocol-unit monetary fields to
// user-facing QC without mutating canonical block/transaction data.
func newBlockDetailPresentationV1(
	chain explorerAmountConverterV1,
	block *blockchain.Block,
) *explorerBlockV1 {
	if block == nil {
		return nil
	}

	var transactions []*explorerTransactionV1
	if block.Transactions != nil {
		transactions =
			make(
				[]*explorerTransactionV1,
				len(block.Transactions),
			)

		for txIndex, tx := range block.Transactions {
			if tx == nil {
				continue
			}

			var outputs []explorerTransactionOutputV1

			if tx.Outputs != nil {
				outputs =
					make(
						[]explorerTransactionOutputV1,
						len(tx.Outputs),
					)

				for outputIndex, output := range tx.Outputs {
					outputs[outputIndex] =
						explorerTransactionOutputV1{
							Amount: chain.
								TransactionAmountQCAtHeightV1(
									float64(output.Amount),
									block.Index,
								),
							PubKeyHash: output.PubKeyHash,
						}
				}
			}

			transactions[txIndex] =
				&explorerTransactionV1{
					ID:        tx.ID,
					Inputs:    tx.Inputs,
					Outputs:   outputs,
					Timestamp: tx.Timestamp,
					Sender:    tx.Sender,
					Amount: chain.
						TransactionAmountQCAtHeightV1(
							tx.Amount,
							block.Index,
						),
				}
		}
	}

	return &explorerBlockV1{
		Index:        block.Index,
		Timestamp:    block.Timestamp,
		Transactions: transactions,
		PrevHash:     block.PrevHash,
		Hash:         block.Hash,
		Nonce:        block.Nonce,
		Miner:        block.Miner,
		Difficulty:   block.Difficulty,
		Metadata:     block.Metadata,
	}
}
