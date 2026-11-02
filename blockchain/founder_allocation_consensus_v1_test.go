package blockchain

import (
	"bytes"
	"testing"
	"time"

	"quantumcoin/config"
	"quantumcoin/wallet"
)

func TestFounderAllocationAndVestingBindingV1(t *testing.T) {
	original := config.Current()
	t.Cleanup(func() {
		config.Set(original)
	})

	liquidWallet := wallet.NewWallet()
	lockedWallet := wallet.NewWallet()

	if liquidWallet.GetAddress() == lockedWallet.GetAddress() {
		t.Fatalf("generated founder addresses are unexpectedly equal")
	}

	cfg := config.Default()
	cfg.PremineLiquidAddress = liquidWallet.GetAddress()
	cfg.PremineLockedAddress = lockedWallet.GetAddress()

	if err := config.SetValidated(cfg); err != nil {
		t.Fatalf("set founder config: %v", err)
	}

	chain := NewBlockchain(
		cfg.InitialReward,
		cfg.TotalSupply,
	)
	if chain == nil || len(chain.Blocks) != 1 {
		t.Fatalf("expected canonical genesis chain")
	}

	if !FounderVestingReadyV1() {
		t.Fatalf("founder vesting should be active")
	}

	liquidHash, err := wallet.Base58DecodeAddressSafe(
		cfg.PremineLiquidAddress,
	)
	if err != nil {
		t.Fatalf("decode liquid address: %v", err)
	}

	lockedHash, err := wallet.Base58DecodeAddressSafe(
		cfg.PremineLockedAddress,
	)
	if err != nil {
		t.Fatalf("decode locked address: %v", err)
	}

	liquidTotal := 0
	lockedTotal := 0
	lockedTxID := []byte(nil)
	lockedOutIndex := -1

	for _, tx := range chain.Blocks[0].Transactions {
		if tx == nil {
			continue
		}

		for outputIndex, output := range tx.Outputs {
			switch {
			case bytes.Equal(output.PubKeyHash, liquidHash):
				liquidTotal += output.Amount

			case bytes.Equal(output.PubKeyHash, lockedHash):
				lockedTotal += output.Amount
				lockedTxID = append([]byte(nil), tx.ID...)
				lockedOutIndex = outputIndex
			}
		}
	}

	if liquidTotal != 255_000 {
		t.Fatalf(
			"liquid founder allocation: got %d want %d",
			liquidTotal,
			255_000,
		)
	}

	if lockedTotal != FounderLockedAllocationV1 {
		t.Fatalf(
			"locked founder allocation: got %d want %d",
			lockedTotal,
			FounderLockedAllocationV1,
		)
	}

	if liquidTotal+lockedTotal != 1_275_000 {
		t.Fatalf(
			"founder total allocation: got %d want %d",
			liquidTotal+lockedTotal,
			1_275_000,
		)
	}

	if len(lockedTxID) == 0 || lockedOutIndex < 0 {
		t.Fatalf("locked founder outpoint not found")
	}

	makeSpend := func(release int) *Transaction {
		return &Transaction{
			Inputs: []TransactionInput{
				{
					TxID:     append([]byte(nil), lockedTxID...),
					OutIndex: lockedOutIndex,
				},
			},
			Outputs: []TransactionOutput{
				{
					Amount: FounderLockedAllocationV1 - release,
					PubKeyHash: append(
						[]byte(nil),
						lockedHash...,
					),
				},
				{
					Amount: release,
					PubKeyHash: append(
						[]byte(nil),
						liquidHash...,
					),
				},
			},
			Timestamp: time.Unix(
				CanonicalGenesisUnixV1,
				0,
			).UTC(),
		}
	}

	genesisTime := CanonicalGenesisTimeV1()

	if err := chain.ValidateFounderVestingSpendV1(
		makeSpend(1),
		genesisTime.AddDate(1, 0, 0).Unix(),
	); err == nil {
		t.Fatalf("expected pre-unlock founder spend rejection")
	}

	if err := chain.ValidateFounderVestingSpendV1(
		makeSpend(FounderMonthlyUnlockQCV1),
		genesisTime.AddDate(1, 1, 0).Unix(),
	); err != nil {
		t.Fatalf("first unlocked tranche rejected: %v", err)
	}

	if err := chain.ValidateFounderVestingSpendV1(
		makeSpend(FounderMonthlyUnlockQCV1+1),
		genesisTime.AddDate(1, 1, 0).Unix(),
	); err == nil {
		t.Fatalf("expected first-tranche overspend rejection")
	}

	if got := FounderUnlockedAtUnixV1(
		genesisTime.Unix(),
		genesisTime.AddDate(5, 0, 0).Unix(),
	); got != FounderLockedAllocationV1 {
		t.Fatalf(
			"fully vested amount: got %d want %d",
			got,
			FounderLockedAllocationV1,
		)
	}
}
