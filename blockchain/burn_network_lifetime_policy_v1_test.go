package blockchain

import (
	"testing"

	"quantumcoin/wallet"
)

func TestValidateCanonicalBurnNetworkLifetimeLimitV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	// Canonical network history: 4 atoms.
	bc.Blocks = append(
		bc.Blocks,
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				historicalBurnForOwnerV1(
					t,
					owner,
					4,
					0x81,
				),
			},
		},
	)

	tests := []struct {
		name      string
		burnAtoms int
		maxAtoms  int64
		wantErr   bool
	}{
		{
			name:      "below network lifetime limit",
			burnAtoms: 5,
			maxAtoms:  10,
			wantErr:   false,
		},
		{
			name:      "exact network lifetime limit",
			burnAtoms: 6,
			maxAtoms:  10,
			wantErr:   false,
		},
		{
			name:      "above network lifetime limit",
			burnAtoms: 7,
			maxAtoms:  10,
			wantErr:   true,
		},
	}

	for index, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				tx :=
					historicalBurnForOwnerV1(
						t,
						owner,
						test.burnAtoms,
						byte(0x90+index),
					)

				err :=
					bc.validateCanonicalBurnNetworkLifetimeLimitAtHeightV1(
						tx,
						1,
						0,
						test.maxAtoms,
					)

				if test.wantErr && err == nil {
					t.Fatalf(
						"burn=%d historical=4 max=%d unexpectedly accepted",
						test.burnAtoms,
						test.maxAtoms,
					)
				}

				if !test.wantErr && err != nil {
					t.Fatalf(
						"burn=%d historical=4 max=%d rejected: %v",
						test.burnAtoms,
						test.maxAtoms,
						err,
					)
				}
			},
		)
	}
}

func TestValidateCanonicalBurnNetworkLifetimeLimitIncludesCandidatePriorBurnsV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	tx :=
		historicalBurnForOwnerV1(
			t,
			owner,
			4,
			0xA1,
		)

	// Canonical history = 0.
	// Earlier burns in the same candidate sequence = 7.
	// Current burn = 4.
	// Network lifetime maximum fixture = 10.
	// 7 + 4 = 11 => reject.
	if err :=
		bc.validateCanonicalBurnNetworkLifetimeLimitAtHeightV1(
			tx,
			1,
			7,
			10,
		); err == nil {
		t.Fatal(
			"candidate sequence network lifetime overflow unexpectedly accepted",
		)
	}
}

func TestValidateCanonicalBurnNetworkLifetimeLimitAcrossOwnersV1(
	t *testing.T,
) {
	bc, firstOwner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	secondOwner := wallet.NewWallet()
	if secondOwner == nil {
		t.Fatal(
			"create second owner wallet returned nil",
		)
	}

	// First owner already burned 6 atoms on canonical chain.
	bc.Blocks = append(
		bc.Blocks,
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				historicalBurnForOwnerV1(
					t,
					firstOwner,
					6,
					0xB1,
				),
			},
		},
	)

	// A completely different wallet tries to burn another 5 atoms.
	// Global maximum fixture is 10, so 6 + 5 = 11 must fail.
	tx :=
		historicalBurnForOwnerV1(
			t,
			secondOwner,
			5,
			0xB2,
		)

	if err :=
		bc.validateCanonicalBurnNetworkLifetimeLimitAtHeightV1(
			tx,
			1,
			0,
			10,
		); err == nil {
		t.Fatal(
			"cross-owner network lifetime limit bypass unexpectedly accepted",
		)
	}
}

func TestValidateCanonicalBurnNetworkLifetimeLimitRejectsInvalidPolicyV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	tx :=
		historicalBurnForOwnerV1(
			t,
			owner,
			1,
			0xC1,
		)

	for _, maxAtoms := range []int64{
		0,
		-1,
	} {
		if err :=
			bc.validateCanonicalBurnNetworkLifetimeLimitAtHeightV1(
				tx,
				1,
				0,
				maxAtoms,
			); err == nil {
			t.Fatalf(
				"invalid network lifetime max=%d unexpectedly accepted",
				maxAtoms,
			)
		}
	}
}
