package blockchain

import (
	"testing"
)

func TestValidateCanonicalBurnPerTransactionLimitV1(
	t *testing.T,
) {
	bc, _ :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	tests := []struct {
		name       string
		burnAtoms  int
		maxAtoms   int64
		wantReject bool
	}{
		{
			name:       "below limit",
			burnAtoms:  9,
			maxAtoms:   10,
			wantReject: false,
		},
		{
			name:       "exact limit",
			burnAtoms:  10,
			maxAtoms:   10,
			wantReject: false,
		},
		{
			name:       "above limit",
			burnAtoms:  11,
			maxAtoms:   10,
			wantReject: true,
		},
	}

	for _, tc := range tests {
		t.Run(
			tc.name,
			func(t *testing.T) {
				tx := &Transaction{
					Inputs: []TransactionInput{
						{
							TxID:     []byte{0x01},
							OutIndex: 0,
						},
					},
					Outputs: []TransactionOutput{
						NewCanonicalBurnOutputV1(
							tc.burnAtoms,
						),
					},
				}

				err :=
					bc.validateCanonicalBurnPerTransactionLimitAtHeightV1(
						tx,
						1,
						tc.maxAtoms,
					)

				if tc.wantReject {
					if err == nil {
						t.Fatalf(
							"burn=%d max=%d unexpectedly accepted",
							tc.burnAtoms,
							tc.maxAtoms,
						)
					}
					return
				}

				if err != nil {
					t.Fatalf(
						"burn=%d max=%d rejected: %v",
						tc.burnAtoms,
						tc.maxAtoms,
						err,
					)
				}
			},
		)
	}
}

func TestValidateCanonicalBurnPerTransactionLimitRejectsInvalidPolicyV1(
	t *testing.T,
) {
	bc, _ :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	tx := &Transaction{
		Inputs: []TransactionInput{
			{
				TxID:     []byte{0x01},
				OutIndex: 0,
			},
		},
		Outputs: []TransactionOutput{
			NewCanonicalBurnOutputV1(
				1,
			),
		},
	}

	for _, maxAtoms := range []int64{
		0,
		-1,
	} {
		if err :=
			bc.validateCanonicalBurnPerTransactionLimitAtHeightV1(
				tx,
				1,
				maxAtoms,
			); err == nil {
			t.Fatalf(
				"invalid maxAtoms=%d unexpectedly accepted",
				maxAtoms,
			)
		}
	}
}
