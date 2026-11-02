package token

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestFungiblePersistenceV1RoundTripsBothModels(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := canonicalAddressFixture(
		"owner",
	)
	holder := canonicalAddressFixture(
		"holder",
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
		holder,
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

	cases := []struct {
		name       string
		subject    FungibleTokenReadV1
		model      FungibleTokenModelV1
		balanceKey string
		balance    uint64
	}{
		{
			name:       "controlled emission",
			subject:    controlled,
			model:      FungibleTokenModelControlledEmissionV1,
			balanceKey: holder,
			balance:    25,
		},
		{
			name:       "fixed supply",
			subject:    fixed,
			model:      FungibleTokenModelFixedSupplyV1,
			balanceKey: owner,
			balance:    100,
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				raw, err := MarshalFungibleTokenV1(
					testCase.subject,
				)
				if err != nil {
					t.Fatalf(
						"marshal fungible token: %v",
						err,
					)
				}

				var envelope map[string]json.RawMessage
				if err := json.Unmarshal(
					raw,
					&envelope,
				); err != nil {
					t.Fatalf("decode envelope: %v", err)
				}

				if len(envelope) != 3 {
					t.Fatalf(
						"envelope field count = %d, want 3",
						len(envelope),
					)
				}

				loaded, err :=
					UnmarshalFungibleTokenV1(raw)

				if err != nil {
					t.Fatalf(
						"unmarshal fungible token: %v",
						err,
					)
				}

				if loaded.FungibleModelV1() !=
					testCase.model {
					t.Fatalf(
						"loaded model = %q, want %q",
						loaded.FungibleModelV1(),
						testCase.model,
					)
				}

				if loaded.BalanceOf(
					testCase.balanceKey,
				) != testCase.balance {
					t.Fatalf(
						"loaded balance = %d, want %d",
						loaded.BalanceOf(
							testCase.balanceKey,
						),
						testCase.balance,
					)
				}
			},
		)
	}
}

func TestFungiblePersistenceV1PreservesHistoricalInnerJSON(
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

	controlledRaw, err := json.Marshal(controlled)
	if err != nil {
		t.Fatalf("marshal controlled token: %v", err)
	}

	fixedRaw, err := json.Marshal(fixed)
	if err != nil {
		t.Fatalf("marshal fixed token: %v", err)
	}

	var controlledObject map[string]json.RawMessage
	var fixedObject map[string]json.RawMessage

	if err := json.Unmarshal(
		controlledRaw,
		&controlledObject,
	); err != nil {
		t.Fatalf("decode controlled JSON: %v", err)
	}

	if err := json.Unmarshal(
		fixedRaw,
		&fixedObject,
	); err != nil {
		t.Fatalf("decode fixed JSON: %v", err)
	}

	if _, exists := controlledObject["model"]; exists {
		t.Fatal("historical Token JSON gained model field")
	}

	if _, exists := fixedObject["Model"]; exists {
		t.Fatal("historical QC20 JSON gained Model field")
	}

	if _, exists := controlledObject["symbol"]; !exists {
		t.Fatal("historical Token symbol key changed")
	}

	if _, exists := fixedObject["Symbol"]; !exists {
		t.Fatal("historical QC20 Symbol key changed")
	}
}

func TestFungiblePersistenceV1RejectsAmbiguousModels(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

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

	controlledState, err := json.Marshal(controlled)
	if err != nil {
		t.Fatal(err)
	}

	fixedState, err := json.Marshal(fixed)
	if err != nil {
		t.Fatal(err)
	}

	versionTwo, err := json.Marshal(
		fungiblePersistenceEnvelopeV1{
			Version: 2,
			Model:   FungibleTokenModelControlledEmissionV1,
			State:   controlledState,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	unknownModel := []byte(
		`{"version":1,"model":"UNKNOWN","state":{}}`,
	)

	duplicateModel := []byte(
		`{"version":1,"model":"TOKEN","model":"QC20","state":{}}`,
	)

	crossModel, err := json.Marshal(
		fungiblePersistenceEnvelopeV1{
			Version: FungiblePersistenceVersionV1,
			Model:   FungibleTokenModelControlledEmissionV1,
			State:   fixedState,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		raw      []byte
		sentinel error
	}{
		{
			name:     "unsupported version",
			raw:      versionTwo,
			sentinel: ErrFungiblePersistenceVersionV1,
		},
		{
			name:     "unknown model",
			raw:      unknownModel,
			sentinel: ErrFungiblePersistenceModelV1,
		},
		{
			name:     "duplicate model",
			raw:      duplicateModel,
			sentinel: ErrFungiblePersistenceEnvelopeV1,
		},
		{
			name:     "cross-model state",
			raw:      crossModel,
			sentinel: ErrFungiblePersistenceIntegrityV1,
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				loaded, err :=
					UnmarshalFungibleTokenV1(
						testCase.raw,
					)

				if loaded != nil {
					t.Fatal("rejected envelope returned a token")
				}

				if !errors.Is(
					err,
					testCase.sentinel,
				) {
					t.Fatalf(
						"error = %v, want %v",
						err,
						testCase.sentinel,
					)
				}
			},
		)
	}
}

func qc20StateFixture(
	overrides map[string]any,
) []byte {
	state := map[string]any{
		"Name":        "Fixed Token",
		"Symbol":      "FIX",
		"Owner":       "owner",
		"MaxSupply":   uint64(100),
		"TotalSupply": uint64(100),
		"Decimals":    uint8(8),
		"Balances": map[string]uint64{
			"owner": 100,
		},
		"Allowances": map[string]map[string]uint64{},
	}

	for key, value := range overrides {
		state[key] = value
	}

	raw, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}

	return raw
}

func TestQC20PersistenceIntegrityV1RejectsAtomically(
	t *testing.T,
) {
	subject := NewQC20TokenWithOwner(
		"Fixed Token",
		"FIX",
		100,
		8,
		"owner",
	)

	cases := []struct {
		name      string
		overrides map[string]any
	}{
		{
			name: "supply above cap",
			overrides: map[string]any{
				"MaxSupply":   uint64(99),
				"TotalSupply": uint64(100),
			},
		},
		{
			name: "balance mismatch",
			overrides: map[string]any{
				"Balances": map[string]uint64{
					"owner": 99,
				},
			},
		},
		{
			name: "issued supply without owner",
			overrides: map[string]any{
				"Owner": "",
			},
		},
		{
			name: "invalid symbol",
			overrides: map[string]any{
				"Symbol": " ",
			},
		},
		{
			name: "invalid balance address",
			overrides: map[string]any{
				"Balances": map[string]uint64{
					" ": 100,
				},
			},
		},
		{
			name: "invalid allowance owner",
			overrides: map[string]any{
				"Allowances": map[string]map[string]uint64{
					" ": {
						"spender": 1,
					},
				},
			},
		},
		{
			name: "invalid nonce caller",
			overrides: map[string]any{
				"AuthorizationNonces": map[string]uint64{
					" ": 1,
				},
			},
		},
	}

	for _, testCase := range cases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				err := json.Unmarshal(
					qc20StateFixture(
						testCase.overrides,
					),
					subject,
				)

				if !errors.Is(
					err,
					ErrQC20PersistenceIntegrityV1,
				) {
					t.Fatalf(
						"error = %v, want QC20 integrity error",
						err,
					)
				}

				if subject.Name() != "Fixed Token" ||
					subject.Symbol() != "FIX" ||
					subject.Owner() != "owner" ||
					subject.MaxSupply() != 100 ||
					subject.TotalSupply() != 100 ||
					subject.BalanceOf("owner") != 100 {
					t.Fatal(
						"invalid load changed existing QC20 state",
					)
				}
			},
		)
	}
}

func TestQC20PersistenceIntegrityV1AcceptsLegacyOwnerlessState(
	t *testing.T,
) {
	raw := qc20StateFixture(
		map[string]any{
			"Owner":       "",
			"MaxSupply":   uint64(1000),
			"TotalSupply": uint64(0),
			"Balances":    map[string]uint64{},
		},
	)

	var subject QC20Token

	if err := json.Unmarshal(raw, &subject); err != nil {
		t.Fatalf("load legacy ownerless QC20: %v", err)
	}

	if subject.Owner() != "" ||
		subject.TotalSupply() != 0 ||
		subject.MaxSupply() != 1000 {
		t.Fatal("legacy ownerless QC20 state changed")
	}
}
