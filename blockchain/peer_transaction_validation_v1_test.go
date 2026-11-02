package blockchain

import (
	"strings"
	"testing"
	"time"
)

func testPeerTxCoinbaseV1(amount int) *Transaction {
	tx := &Transaction{
		ID:     nil,
		Inputs: []TransactionInput{},
		Outputs: []TransactionOutput{
			{Amount: amount, PubKeyHash: []byte("peer-miner")},
		},
		Timestamp: time.Now(),
		Sender:    "COINBASE",
		Amount:    float64(amount),
	}
	tx.ID = tx.Hash()
	return tx
}

func TestValidatePeerTransactionStrictV1RejectsNilTx(t *testing.T) {
	bc := NewBlockchain(1, 0)

	err := bc.ValidatePeerTransactionStrictV1(nil)
	if err == nil {
		t.Fatalf("expected nil tx rejection")
	}
}

func TestValidatePeerTransactionStrictV1RejectsCoinbase(t *testing.T) {
	bc := NewBlockchain(1, 0)

	err := bc.ValidatePeerTransactionStrictV1(testPeerTxCoinbaseV1(1))
	if err == nil {
		t.Fatalf("expected peer coinbase rejection")
	}
	if !strings.Contains(err.Error(), "coinbase") {
		t.Fatalf("expected coinbase rejection, got: %v", err)
	}
}

func TestValidatePeerTransactionStrictV1RejectsUnsignedNormalTx(t *testing.T) {
	bc := NewBlockchain(1, 0)

	tx := &Transaction{
		ID: []byte("peer-unsigned-tx"),
		Inputs: []TransactionInput{
			{TxID: []byte("missing-utxo"), OutIndex: 0},
		},
		Outputs: []TransactionOutput{
			{Amount: 1, PubKeyHash: []byte("receiver")},
		},
		Timestamp: time.Now(),
		Sender:    "QC_FAKE_SENDER",
		Amount:    1,
	}

	tx.ID = tx.Hash()
	err := bc.ValidatePeerTransactionStrictV1(tx)
	if err == nil {
		t.Fatalf("expected unsigned peer tx rejection")
	}
	if !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature rejection, got: %v", err)
	}
}
