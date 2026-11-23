package blockchain

import (
	"fmt"
	"strings"
	"time"

	"quantumcoin/wallet"
)

func newCoinbaseTxWithRewardV1(
	miner string,
	reward int,
	timestamp time.Time,
) (*Transaction, error) {
	if strings.TrimSpace(miner) == "" {
		return nil, ErrMinerAddressEmpty
	}

	if reward < 0 {
		return nil, fmt.Errorf(
			"coinbase reward cannot be negative",
		)
	}

	transaction := &Transaction{
		ID:     nil,
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{
				Amount: reward,
				PubKeyHash: wallet.Base58DecodeAddress(
					miner,
				),
			},
		},
		Timestamp: timestamp.UTC(),
		Sender:    "COINBASE",
		Amount:    float64(reward),
	}
	transaction.ID = transaction.Hash()

	return transaction, nil
}
