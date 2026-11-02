package token

import (
	"reflect"
	"testing"
)

func TestFungibleModelContractV1IdentifiesRoles(
	t *testing.T,
) {
	controlled := New(
		"MINE",
		"Mineable Token",
		8,
		1000,
		"owner",
	)

	fixed := NewQC20TokenWithOwner(
		"Fixed Token",
		"FIX",
		100,
		8,
		"owner",
	)

	cases := []struct {
		name    string
		subject FungibleTokenReadV1
		model   FungibleTokenModelV1
		policy  FungibleSupplyPolicyV1
	}{
		{
			name:    "controlled emission",
			subject: controlled,
			model:   FungibleTokenModelControlledEmissionV1,
			policy:  FungibleSupplyPolicyControlledEmissionV1,
		},
		{
			name:    "fixed supply",
			subject: fixed,
			model:   FungibleTokenModelFixedSupplyV1,
			policy:  FungibleSupplyPolicyFixedSupplyV1,
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				if testCase.subject.FungibleModelV1() !=
					testCase.model {
					t.Fatalf(
						"model = %q, want %q",
						testCase.subject.FungibleModelV1(),
						testCase.model,
					)
				}

				if testCase.subject.FungibleSupplyPolicyV1() !=
					testCase.policy {
					t.Fatalf(
						"policy = %q, want %q",
						testCase.subject.FungibleSupplyPolicyV1(),
						testCase.policy,
					)
				}

				if testCase.subject.Symbol() == "" ||
					testCase.subject.Name() == "" ||
					testCase.subject.Owner() == "" {
					t.Fatal("common read contract returned empty identity")
				}
			},
		)
	}
}

func TestFungibleModelContractV1IsReadOnly(
	t *testing.T,
) {
	contract := reflect.TypeOf(
		(*FungibleTokenReadV1)(nil),
	).Elem()

	for _, forbidden := range []string{
		"Mint",
		"MintByOwner",
		"Burn",
		"Transfer",
		"TransferFrom",
		"Approve",
	} {
		if _, exists := contract.MethodByName(
			forbidden,
		); exists {
			t.Fatalf(
				"read contract exposes mutator %q",
				forbidden,
			)
		}
	}
}

func TestFungibleModelContractV1SnapshotsAreDefensive(
	t *testing.T,
) {
	owner := canonicalAddressFixture(
		"owner",
	)

	controlled := New(
		"MINE",
		"Mineable Token",
		8,
		1000,
		owner,
	)

	if err := controlled.Mint(
		owner,
		owner,
		25,
	); err != nil {
		t.Fatalf("mint controlled token: %v", err)
	}

	fixed := NewQC20TokenWithOwner(
		"Fixed Token",
		"FIX",
		100,
		8,
		owner,
	)

	subjects := []FungibleTokenReadV1{
		controlled,
		fixed,
	}

	for _, subject := range subjects {
		before := subject.BalanceOf(owner)
		snapshot := subject.BalancesSnapshotV1()
		snapshot[owner] = 0
		snapshot["attacker"] = 999

		if subject.BalanceOf(owner) != before {
			t.Fatal("snapshot changed internal owner balance")
		}

		if subject.BalanceOf("attacker") != 0 {
			t.Fatal("snapshot created an internal attacker balance")
		}
	}
}
