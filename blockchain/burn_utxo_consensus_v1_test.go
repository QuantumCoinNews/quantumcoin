package blockchain

import (
	"strings"
	"testing"
	"time"

	"quantumcoin/wallet"
)

// QMA-0062:
// A canonical burn output must be rejected as an input by the canonical
// UTXO validator even when an attacker supplies an otherwise valid
// wallet key and signature.
func TestCanonicalUTXOValidationRejectsBurnOutputSpendV1(t *testing.T) {
	chain, attacker, receiver, _ := newFeeConsensusTestStateV1(t)

	funding := chain.Blocks[0].Transactions[0]
	if funding == nil || len(funding.Outputs) == 0 {
		t.Fatal("fixture funding transaction/output missing")
	}

	funding.Outputs[0] = NewCanonicalBurnOutputV1(
		funding.Outputs[0].Amount,
	)
	funding.ID = funding.Hash()

	chain.UpdateUTXOSet()

	spend := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     append([]byte(nil), funding.ID...),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount:     1,
				PubKeyHash: wallet.HashPubKey(receiver.PublicKey),
			},
		},
		Timestamp: time.Unix(
			CanonicalGenesisUnixV1+90,
			0,
		).UTC(),
		Sender: attacker.GetAddress(),
		Amount: 1,
	}

	spend.ID = spend.Hash()

	if err := spend.Sign(attacker.PrivateKey); err != nil {
		t.Fatalf("sign attempted burn-output spend: %v", err)
	}

	_, err := chain.ValidateTransactionUTXOV1(
		spend,
		false,
	)

	if err == nil {
		t.Fatal(
			"canonical burn output must never be spendable",
		)
	}

	if !strings.Contains(
		err.Error(),
		"canonical burn output is unspendable",
	) {
		t.Fatalf(
			"expected canonical burn consensus rejection, got %v",
			err,
		)
	}
}
