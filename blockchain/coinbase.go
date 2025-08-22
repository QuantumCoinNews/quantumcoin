package blockchain

import (
	"errors"
	"strings"
	"time"

	"quantumcoin/wallet"
)

func newCoinbaseTx(miner string) (*Transaction, error) {
	if strings.TrimSpace(miner) == "" {
		return nil, errors.New("miner address empty")
	}
	reward := GetCurrentReward()
	tx := &Transaction{
		ID:     nil,
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{Amount: reward, PubKeyHash: wallet.Base58DecodeAddress(miner)},
		},
		Timestamp: time.Now(),
		Sender:    "COINBASE",
		Amount:    float64(reward),
	}
	tx.ID = tx.Hash()
	return tx, nil
}
