package blockchain

import (
	"testing"

	"quantumcoin/wallet"
)

func TestValidateCanonicalBurnOwnerLifetimeLimitV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	ownerKey :=
		wallet.HashPubKey(
			owner.PublicKey,
		)

	// Historical canonical burn: 4 atoms.
	bc.Blocks = append(
		bc.Blocks,
		&Block{
			Index: 1,
			Transactions: []*Transaction{
				historicalBurnForOwnerV1(
					t,
					owner,
					4,
					0x51,
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
			name:      "below lifetime limit",
			burnAtoms: 5,
			maxAtoms:  10,
			wantErr:   false,
		},
		{
			name:      "exact lifetime limit",
			burnAtoms: 6,
			maxAtoms:  10,
			wantErr:   false,
		},
		{
			name:      "above lifetime limit",
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
						byte(0x60+index),
					)

				err :=
					bc.validateCanonicalBurnOwnerLifetimeLimitAtHeightV1(
						tx,
						1,
						ownerKey,
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

func TestValidateCanonicalBurnOwnerLifetimeLimitIncludesCandidatePriorBurnsV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	ownerKey :=
		wallet.HashPubKey(
			owner.PublicKey,
		)

	tx :=
		historicalBurnForOwnerV1(
			t,
			owner,
			4,
			0x71,
		)

	// Canonical history = 0.
	// Earlier burns by this owner in the same candidate sequence = 7.
	// Candidate burn = 4.
	// Lifetime maximum fixture = 10.
	// 7 + 4 = 11 => reject.
	if err :=
		bc.validateCanonicalBurnOwnerLifetimeLimitAtHeightV1(
			tx,
			1,
			ownerKey,
			7,
			10,
		); err == nil {
		t.Fatal(
			"candidate sequence lifetime overflow unexpectedly accepted",
		)
	}
}

func TestValidateCanonicalBurnOwnerLifetimeLimitRejectsInvalidPolicyV1(
	t *testing.T,
) {
	bc, owner :=
		atomSpendSelectionTestChainV1(
			t,
			1,
		)

	ownerKey :=
		wallet.HashPubKey(
			owner.PublicKey,
		)

	tx :=
		historicalBurnForOwnerV1(
			t,
			owner,
			1,
			0x72,
		)

	for _, maxAtoms := range []int64{
		0,
		-1,
	} {
		if err :=
			bc.validateCanonicalBurnOwnerLifetimeLimitAtHeightV1(
				tx,
				1,
				ownerKey,
				0,
				maxAtoms,
			); err == nil {
			t.Fatalf(
				"invalid lifetime max=%d unexpectedly accepted",
				maxAtoms,
			)
		}
	}
}
