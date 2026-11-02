package blockchain

import "testing"

func coinbaseBridgeTestCoinbaseV1(amounts ...int) *Transaction {
	outputs := make([]TransactionOutput, len(amounts))

	for index, amount := range amounts {
		outputs[index] = TransactionOutput{
			Amount: amount,
		}
	}

	return &Transaction{
		Inputs:  []TransactionInput{},
		Outputs: outputs,
		Sender:  "COINBASE",
	}
}

func coinbaseBridgeTestRegularTransactionV1() *Transaction {
	return &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     []byte("previous-transaction"),
				OutIndex: 0,
			},
		},
	}
}

func TestBlockCoinbaseTotalV1ConsensusBridge(t *testing.T) {
	tests := []struct {
		name          string
		block         *Block
		allowMultiple bool
		wantCount     int
		wantTotal     int
		wantError     bool
	}{
		{
			name: "normal block accepts first and single coinbase",
			block: &Block{
				Transactions: []*Transaction{
					coinbaseBridgeTestCoinbaseV1(50, 3),
					coinbaseBridgeTestRegularTransactionV1(),
				},
			},
			wantCount: 1,
			wantTotal: 53,
		},
		{
			name: "normal block rejects coinbase outside first position",
			block: &Block{
				Transactions: []*Transaction{
					coinbaseBridgeTestRegularTransactionV1(),
					coinbaseBridgeTestCoinbaseV1(50),
				},
			},
			wantError: true,
		},
		{
			name: "normal block rejects multiple coinbase transactions",
			block: &Block{
				Transactions: []*Transaction{
					coinbaseBridgeTestCoinbaseV1(25),
					coinbaseBridgeTestCoinbaseV1(25),
				},
			},
			wantError: true,
		},
		{
			name: "genesis accepts multiple allocation transactions",
			block: &Block{
				Transactions: []*Transaction{
					coinbaseBridgeTestCoinbaseV1(100),
					coinbaseBridgeTestCoinbaseV1(200),
				},
			},
			allowMultiple: true,
			wantCount:     2,
			wantTotal:     300,
		},
		{
			name:      "nil block is rejected",
			block:     nil,
			wantError: true,
		},
		{
			name: "empty block is rejected",
			block: &Block{
				Transactions: nil,
			},
			wantError: true,
		},
		{
			name: "nil transaction is rejected",
			block: &Block{
				Transactions: []*Transaction{
					nil,
				},
			},
			wantError: true,
		},
		{
			name: "block without coinbase is rejected",
			block: &Block{
				Transactions: []*Transaction{
					coinbaseBridgeTestRegularTransactionV1(),
				},
			},
			wantError: true,
		},
		{
			name: "zero coinbase output is rejected",
			block: &Block{
				Transactions: []*Transaction{
					coinbaseBridgeTestCoinbaseV1(0),
				},
			},
			wantError: true,
		},
		{
			name: "negative coinbase output is rejected",
			block: &Block{
				Transactions: []*Transaction{
					coinbaseBridgeTestCoinbaseV1(-1),
				},
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			count, total, err := blockCoinbaseTotalV1(
				test.block,
				test.allowMultiple,
			)

			if test.wantError {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}

			if count != test.wantCount || total != test.wantTotal {
				t.Fatalf(
					"result count=%d total=%d, want count=%d total=%d",
					count,
					total,
					test.wantCount,
					test.wantTotal,
				)
			}
		})
	}
}
