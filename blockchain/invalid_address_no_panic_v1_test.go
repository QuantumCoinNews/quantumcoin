package blockchain

import "testing"

func TestGetBalanceInvalidAddressDoesNotPanicV1(t *testing.T) {
	bc := NewBlockchain(25500000, 1)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetBalance/GetSpendableBalance must not panic on invalid address: %v", r)
		}
	}()

	_ = bc.GetBalance("not-a-valid-qc-address@@@")
	_ = bc.GetSpendableBalance("not-a-valid-qc-address@@@")
}

func TestNewTransactionInvalidAddressDoesNotPanicV1(t *testing.T) {
	bc := NewBlockchain(25500000, 1)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewTransaction must not panic on invalid address: %v", r)
		}
	}()

	_, _ = NewTransaction("not-a-valid-qc-address@@@", "also-not-valid@@@", 1, bc)
}

func TestTransactionVerifyInvalidSenderDoesNotPanicV1(t *testing.T) {
	tx := &Transaction{
		Sender: "not-a-valid-qc-address@@@",
		Inputs: []TransactionInput{
			{TxID: []byte("x"), PubKey: []byte("bad")},
		},
		Outputs: []TransactionOutput{
			{Amount: 1, PubKeyHash: []byte("bad")},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Transaction.Verify must not panic on invalid sender: %v", r)
		}
	}()

	if tx.Verify() {
		t.Fatalf("expected invalid sender transaction to fail verification")
	}
}
