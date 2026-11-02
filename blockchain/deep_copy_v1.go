package blockchain

func cloneBytesV1(source []byte) []byte {
	if source == nil {
		return nil
	}

	cloned := make([]byte, len(source))
	copy(cloned, source)

	return cloned
}

func cloneTransactionInputV1(
	input TransactionInput,
) TransactionInput {
	return TransactionInput{
		TxID:      cloneBytesV1(input.TxID),
		OutIndex:  input.OutIndex,
		Signature: cloneBytesV1(input.Signature),
		PubKey:    cloneBytesV1(input.PubKey),
	}
}

func cloneTransactionOutputV1(
	output TransactionOutput,
) TransactionOutput {
	return TransactionOutput{
		Amount:     output.Amount,
		PubKeyHash: cloneBytesV1(output.PubKeyHash),
	}
}

func cloneTransactionV1(
	transaction *Transaction,
) *Transaction {
	if transaction == nil {
		return nil
	}

	cloned := &Transaction{
		ID:        cloneBytesV1(transaction.ID),
		Timestamp: transaction.Timestamp,
		Sender:    transaction.Sender,
		Amount:    transaction.Amount,
	}

	if transaction.Inputs != nil {
		cloned.Inputs = make(
			[]TransactionInput,
			len(transaction.Inputs),
		)

		for index := range transaction.Inputs {
			cloned.Inputs[index] =
				cloneTransactionInputV1(
					transaction.Inputs[index],
				)
		}
	}

	if transaction.Outputs != nil {
		cloned.Outputs = make(
			[]TransactionOutput,
			len(transaction.Outputs),
		)

		for index := range transaction.Outputs {
			cloned.Outputs[index] =
				cloneTransactionOutputV1(
					transaction.Outputs[index],
				)
		}
	}

	return cloned
}

func cloneBlockV1(block *Block) *Block {
	if block == nil {
		return nil
	}

	cloned := &Block{
		Index:      block.Index,
		Timestamp:  block.Timestamp,
		PrevHash:   cloneBytesV1(block.PrevHash),
		Hash:       cloneBytesV1(block.Hash),
		Nonce:      block.Nonce,
		Miner:      block.Miner,
		Difficulty: block.Difficulty,
	}

	if block.Transactions != nil {
		cloned.Transactions = make(
			[]*Transaction,
			len(block.Transactions),
		)

		for index := range block.Transactions {
			cloned.Transactions[index] =
				cloneTransactionV1(
					block.Transactions[index],
				)
		}
	}

	if block.NFTMintRecordsV1 != nil {
		cloned.NFTMintRecordsV1 = make([]NFTMintRecordV1, len(block.NFTMintRecordsV1))
		for i, record := range block.NFTMintRecordsV1 {
			cloned.NFTMintRecordsV1[i] = record
			cloned.NFTMintRecordsV1[i].RecipientPubKeyHash = append([]byte(nil), record.RecipientPubKeyHash...)
		}
	}
	cloned.NFTAwardProofsV1 = cloneNFTAwardProofsV1(block.NFTAwardProofsV1)
	if block.Metadata != nil {
		cloned.Metadata = make(
			map[string]string,
			len(block.Metadata),
		)

		for key, value := range block.Metadata {
			cloned.Metadata[key] = value
		}
	}

	return cloned
}

func cloneBlocksV1(blocks []*Block) []*Block {
	if blocks == nil {
		return nil
	}

	cloned := make([]*Block, len(blocks))

	for index := range blocks {
		cloned[index] = cloneBlockV1(blocks[index])
	}

	return cloned
}
