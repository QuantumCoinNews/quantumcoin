package blockchain

import (
	"strings"
	"testing"
	"time"

	"quantumcoin/wallet"
)

func TestCanonicalUTXOValidationReturnsVerifiedTotalsV1(t *testing.T) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)

	result, err := chain.ValidateTransactionUTXOV1(spend, false)
	if err != nil {
		t.Fatalf("validate canonical UTXO spend: %v", err)
	}
	if result.InputTotal != 10 || result.OutputTotal != 7 || result.Fee != 3 {
		t.Fatalf("unexpected UTXO result: %+v", result)
	}
}

func TestCanonicalUTXOValidationRejectsMissingPrevoutV1(t *testing.T) {
	chain, sender, _, spend := newFeeConsensusTestStateV1(t)
	spend.Inputs[0].TxID = []byte("missing-prevout")
	spend.ID = spend.Hash()
	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("re-sign missing prevout transaction: %v", err)
	}

	_, err := chain.ValidateTransactionUTXOV1(spend, false)
	if err == nil || !strings.Contains(err.Error(), "missing utxo") {
		t.Fatalf("expected missing UTXO rejection, got %v", err)
	}
}

func TestCanonicalUTXOValidationRejectsWrongOutIndexV1(t *testing.T) {
	chain, sender, _, spend := newFeeConsensusTestStateV1(t)
	spend.Inputs[0].OutIndex = 9
	spend.ID = spend.Hash()
	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("re-sign wrong out-index transaction: %v", err)
	}

	_, err := chain.ValidateTransactionUTXOV1(spend, false)
	if err == nil || !strings.Contains(err.Error(), "missing utxo") {
		t.Fatalf("expected out-index rejection, got %v", err)
	}
}

func TestCanonicalUTXOValidationRejectsOwnershipMismatchV1(t *testing.T) {
	chain, _, receiver, spend := newFeeConsensusTestStateV1(t)
	funding := chain.Blocks[0].Transactions[0]
	funding.Outputs[0].PubKeyHash = wallet.HashPubKey(receiver.PublicKey)
	chain.UpdateUTXOSet()

	if !spend.Verify() {
		t.Fatalf("fixture signature must remain valid before ownership check")
	}

	_, err := chain.ValidateTransactionUTXOV1(spend, false)
	if err == nil || !strings.Contains(err.Error(), "does not unlock utxo") {
		t.Fatalf("expected ownership rejection, got %v", err)
	}
}

func TestCanonicalUTXOValidationRejectsInsufficientInputV1(t *testing.T) {
	chain, sender, receiver, spend := newFeeConsensusTestStateV1(t)
	spend.Outputs[0] = TransactionOutput{
		Amount:     11,
		PubKeyHash: wallet.HashPubKey(receiver.PublicKey),
	}
	spend.Amount = 11
	spend.ID = spend.Hash()
	if err := spend.Sign(sender.PrivateKey); err != nil {
		t.Fatalf("re-sign overspend transaction: %v", err)
	}

	_, err := chain.ValidateTransactionUTXOV1(spend, false)
	if err == nil || !strings.Contains(err.Error(), "insufficient input amount") {
		t.Fatalf("expected input/output sum rejection, got %v", err)
	}
}

func TestCanonicalUTXOValidationAllowsIntraBlockSpendV1(t *testing.T) {
	chain, _, receiver, first := newFeeConsensusTestStateV1(t)
	finalReceiver := wallet.NewWallet()

	second := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     append([]byte(nil), first.ID...),
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			{
				Amount:     5,
				PubKeyHash: wallet.HashPubKey(finalReceiver.PublicKey),
			},
		},
		Timestamp: time.Unix(CanonicalGenesisUnixV1+60, 0).UTC(),
		Sender:    receiver.GetAddress(),
		Amount:    5,
	}
	second.ID = second.Hash()
	if err := second.Sign(receiver.PrivateKey); err != nil {
		t.Fatalf("sign chained transaction: %v", err)
	}

	fees, err := chain.ValidateTransactionsUTXOV1(
		[]*Transaction{first, second},
	)
	if err != nil {
		t.Fatalf("validate chained block spends: %v", err)
	}
	if fees != 5 {
		t.Fatalf("chained fees: got %d want 5", fees)
	}
}

func TestCanonicalUTXOValidationRejectsKnownTransactionIDV1(t *testing.T) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	chain.Blocks = append(chain.Blocks, &Block{
		Index:        1,
		Transactions: []*Transaction{spend},
	})
	chain.UpdateUTXOSet()

	_, err := chain.ValidateTransactionUTXOV1(spend, false)
	if err == nil || !strings.Contains(err.Error(), "duplicate transaction id") {
		t.Fatalf("expected historical transaction ID rejection, got %v", err)
	}
}

func TestCanonicalUTXOValidationRejectsSequenceDoubleSpendV1(t *testing.T) {
	chain, _, _, spend := newFeeConsensusTestStateV1(t)
	duplicate := *spend
	duplicate.Inputs = append([]TransactionInput(nil), spend.Inputs...)
	duplicate.Outputs = append([]TransactionOutput(nil), spend.Outputs...)

	_, err := chain.ValidateTransactionsUTXOV1(
		[]*Transaction{spend, &duplicate},
	)
	if err == nil || (!strings.Contains(err.Error(), "duplicate transaction id") &&
		!strings.Contains(err.Error(), "double spend")) {
		t.Fatalf("expected sequence double-spend rejection, got %v", err)
	}
}
