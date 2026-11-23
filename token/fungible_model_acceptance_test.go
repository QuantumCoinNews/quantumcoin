package token

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestFungibleRawStateCannotDecodeAsAnotherModel(t *testing.T) {
	owner := canonicalAddressFixture("model-owner")
	controlled, err := json.Marshal(New("ZERO", "Zero Token", 8, 100, owner))
	if err != nil {
		t.Fatal(err)
	}
	fixed, err := json.Marshal(NewQC20TokenWithOwner("Zero Token", "ZERO", 0, 8, owner))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("controlled into fixed", func(t *testing.T) {
		var target QC20Token
		if err := json.Unmarshal(controlled, &target); err == nil {
			t.Fatal("raw controlled state became fixed supply")
		}
		if target.Owner() != "" || target.MaxSupply() != 0 {
			t.Fatal("rejected cross-model load changed target")
		}
	})
	t.Run("fixed into controlled", func(t *testing.T) {
		var target Token
		if err := json.Unmarshal(fixed, &target); err == nil {
			t.Fatal("raw fixed state became controlled emission")
		}
		if target.Owner() != "" || target.MaxSupply() != 0 {
			t.Fatal("rejected cross-model load changed target")
		}
	})
}

func TestFungibleCanonicalLoadRejectsNullScalarsAndNestedDuplicates(t *testing.T) {
	t.Setenv("QC_NODE_DIR", t.TempDir())
	owner := canonicalAddressFixture("model-owner")
	for _, model := range []FungibleTokenModelV1{FungibleTokenModelControlledEmissionV1, FungibleTokenModelFixedSupplyV1} {
		t.Run(string(model), func(t *testing.T) {
			var subject FungibleTokenReadV1
			capField, decimalsField, balancesField := "max_supply", "decimals", "balances"
			if model == FungibleTokenModelControlledEmissionV1 {
				subject = New("ZERO", "Zero Token", 8, 100, owner)
			} else {
				subject = NewQC20TokenWithOwner("Zero Token", "ZERO", 0, 8, owner)
				capField, decimalsField, balancesField = "MaxSupply", "Decimals", "Balances"
			}
			raw, err := MarshalFungibleTokenV1(subject)
			if err != nil {
				t.Fatal(err)
			}
			var envelope fungiblePersistenceEnvelopeV1
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Fatal(err)
			}
			state := string(envelope.State)
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(envelope.State, &fields); err != nil {
				t.Fatal(err)
			}
			cases := map[string]string{
				"null cap":          strings.Replace(state, `"`+capField+`":`+string(fields[capField]), `"`+capField+`":null`, 1),
				"null decimals":     strings.Replace(state, `"`+decimalsField+`":8`, `"`+decimalsField+`":null`, 1),
				"duplicate balance": strings.Replace(state, `"`+balancesField+`":{}`, `"`+balancesField+`":{"`+owner+`":9,"`+owner+`":0}`, 1),
			}
			for name, payload := range cases {
				t.Run(name, func(t *testing.T) {
					envelope.State = json.RawMessage(payload)
					data, err := json.Marshal(envelope)
					if err != nil {
						t.Fatal(err)
					}
					loaded, err := UnmarshalFungibleTokenV1(data)
					if loaded != nil || !errors.Is(err, ErrFungiblePersistenceIntegrityV1) {
						t.Fatalf("ambiguous state loaded: %v", err)
					}
				})
			}
		})
	}
}

func TestFungibleCanonicalRoundTripPreservesSupplyPolicy(t *testing.T) {
	t.Setenv("QC_NODE_DIR", t.TempDir())
	owner := canonicalAddressFixture("model-owner")
	for _, subject := range []FungibleTokenReadV1{New("CTRL", "Controlled", 8, 100, owner), NewQC20TokenWithOwner("Fixed", "FIX", 100, 8, owner)} {
		t.Run(string(subject.FungibleModelV1()), func(t *testing.T) {
			raw, err := MarshalFungibleTokenV1(subject)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := UnmarshalFungibleTokenV1(raw)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.FungibleModelV1() != subject.FungibleModelV1() || loaded.FungibleSupplyPolicyV1() != subject.FungibleSupplyPolicyV1() {
				t.Fatal("model or supply policy changed")
			}
			switch token := loaded.(type) {
			case *Token:
				if err := token.Mint(owner, owner, 1); !errors.Is(err, ErrTokenMintDisabled) {
					t.Fatalf("controlled token mint = %v", err)
				}
				if token.TotalSupply() != 0 {
					t.Fatal("rejected controlled mint changed supply")
				}
			case *QC20Token:
				if err := token.Mint(owner, 1); !errors.Is(err, ErrQC20FixedSupplyMintDisabled) {
					t.Fatalf("fixed supply changed: %v", err)
				}
				if token.TotalSupply() != 100 {
					t.Fatal("fixed model changed")
				}
			default:
				t.Fatal("unsupported concrete model returned")
			}
		})
	}
}

func TestFungibleWritersAndRawLoadersRejectInconsistentState(t *testing.T) {
	owner := canonicalAddressFixture("model-owner")
	controlled := New("BAD", "Inconsistent", 8, 100, owner)
	controlled.balances[owner] = 1 // Supply remains zero: invalid state.
	fixed := NewQC20TokenWithOwner("Inconsistent", "BAD", 100, 8, owner)
	fixed.totalSupply = 99 // Balances remain 100: invalid state.
	for _, subject := range []FungibleTokenReadV1{controlled, fixed} {
		t.Run(string(subject.FungibleModelV1()), func(t *testing.T) {
			raw, err := json.Marshal(subject)
			if err != nil {
				t.Fatal(err)
			}
			if data, err := MarshalFungibleTokenV1(subject); err == nil || data != nil {
				t.Error("canonical writer exported inconsistent state")
			}
			switch subject.(type) {
			case *Token:
				var loaded Token
				if err := json.Unmarshal(raw, &loaded); err == nil {
					t.Error("raw controlled loader accepted inconsistent supply")
				}
			case *QC20Token:
				var loaded QC20Token
				if err := json.Unmarshal(raw, &loaded); err == nil {
					t.Error("raw fixed loader accepted inconsistent supply")
				}
			}
		})
	}
}
