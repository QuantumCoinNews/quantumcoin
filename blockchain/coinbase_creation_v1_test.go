package blockchain

import (
	"bytes"
	"testing"
	"time"

	"quantumcoin/wallet"
)

func TestNewCoinbaseTxWithRewardV1CanonicalForm(t *testing.T) {
	miner := wallet.NewWallet()
	timestamp := time.Date(
		2026,
		time.July,
		27,
		12,
		34,
		56,
		0,
		time.FixedZone("test-zone", 3*60*60),
	)

	transaction, err := newCoinbaseTxWithRewardV1(
		miner.GetAddress(),
		53,
		timestamp,
	)
	if err != nil {
		t.Fatalf("create canonical coinbase: %v", err)
	}

	if !transaction.IsCoinbase() {
		t.Fatal("created transaction is not coinbase")
	}

	if len(transaction.Inputs) != 0 {
		t.Fatalf(
			"coinbase input count = %d, want 0",
			len(transaction.Inputs),
		)
	}

	if len(transaction.Outputs) != 1 {
		t.Fatalf(
			"coinbase output count = %d, want 1",
			len(transaction.Outputs),
		)
	}

	output := transaction.Outputs[0]

	if output.Amount != 53 {
		t.Fatalf(
			"coinbase output amount = %d, want 53",
			output.Amount,
		)
	}

	wantPubKeyHash := wallet.Base58DecodeAddress(
		miner.GetAddress(),
	)
	if !bytes.Equal(output.PubKeyHash, wantPubKeyHash) {
		t.Fatal("coinbase output does not belong to miner")
	}

	if transaction.Sender != "COINBASE" {
		t.Fatalf(
			"coinbase sender = %q, want COINBASE",
			transaction.Sender,
		)
	}

	if transaction.Amount != 53 {
		t.Fatalf(
			"coinbase convenience amount = %v, want 53",
			transaction.Amount,
		)
	}

	if !transaction.Timestamp.Equal(timestamp.UTC()) {
		t.Fatalf(
			"coinbase timestamp = %v, want %v",
			transaction.Timestamp,
			timestamp.UTC(),
		)
	}

	if transaction.Timestamp.Location() != time.UTC {
		t.Fatalf(
			"coinbase timestamp location = %v, want UTC",
			transaction.Timestamp.Location(),
		)
	}

	if err := ValidateTransactionIDV1(transaction); err != nil {
		t.Fatalf("coinbase transaction ID is not canonical: %v", err)
	}
}

func TestNewCoinbaseTxWithRewardV1RejectsInvalidArguments(
	t *testing.T,
) {
	tests := []struct {
		name   string
		miner  string
		reward int
	}{
		{
			name:   "empty miner",
			miner:  "",
			reward: 50,
		},
		{
			name:   "whitespace miner",
			miner:  "   ",
			reward: 50,
		},
		{
			name:   "zero reward",
			miner:  wallet.NewWallet().GetAddress(),
			reward: 0,
		},
		{
			name:   "negative reward",
			miner:  wallet.NewWallet().GetAddress(),
			reward: -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transaction, err := newCoinbaseTxWithRewardV1(
				test.miner,
				test.reward,
				time.Now().UTC(),
			)

			if err == nil {
				t.Fatalf(
					"expected creation error, got transaction %+v",
					transaction,
				)
			}

			if transaction != nil {
				t.Fatalf(
					"invalid arguments returned transaction %+v",
					transaction,
				)
			}
		})
	}
}
