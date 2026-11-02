package token

import (
	"encoding/json"
	"errors"
	"testing"

	"quantumcoin/wallet"
)

func TestQC20AuthorizationStateV1PersistsNonceAndState(
	t *testing.T,
) {
	owner := wallet.NewWallet().GetAddress()
	spender := wallet.NewWallet().GetAddress()

	subject := NewQC20TokenWithOwner(
		"Factory Token",
		"FCT",
		1000,
		8,
		owner,
	)

	subject.mu.Lock()
	subject.allowances[owner] =
		map[string]uint64{
			spender: 125,
		}

	commitTokenAuthorizationNonceV1(
		subject.authorizationNonces,
		owner,
		1,
	)
	subject.mu.Unlock()

	raw, err := json.Marshal(subject)
	if err != nil {
		t.Fatalf("marshal QC20 token: %v", err)
	}

	var loaded QC20Token

	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal QC20 token: %v", err)
	}

	next, err := loaded.NextAuthorizationNonceV1(owner)
	if err != nil {
		t.Fatalf("read persisted nonce: %v", err)
	}

	if next != 2 {
		t.Fatalf("next nonce = %d, want 2", next)
	}

	loaded.mu.Lock()
	replayErr := checkTokenAuthorizationNonceV1(
		loaded.authorizationNonces,
		owner,
		1,
	)
	loaded.mu.Unlock()

	if !errors.Is(
		replayErr,
		ErrTokenAuthorizationReplay,
	) {
		t.Fatalf(
			"persisted replay error = %v",
			replayErr,
		)
	}

	if balance := loaded.BalanceOf(owner); balance != 1000 {
		t.Fatalf(
			"loaded owner balance = %d, want 1000",
			balance,
		)
	}

	if allowance := loaded.Allowance(
		owner,
		spender,
	); allowance != 125 {
		t.Fatalf(
			"loaded allowance = %d, want 125",
			allowance,
		)
	}
}

func TestQC20AuthorizationStateV1LoadsLegacyJSON(
	t *testing.T,
) {
	owner := canonicalAddressFixture(
		"legacy-owner",
	)
	spender := canonicalAddressFixture(
		"legacy-spender",
	)

	legacyJSON := []byte(`{
"Name":"Legacy QC20",
"Symbol":"LQC",
"Owner":"` + owner + `",
"MaxSupply":1000,
"TotalSupply":100,
"Decimals":8,
"Balances":{"` + owner + `":100},
"Allowances":{
"` + owner + `":{"` + spender + `":25}
}
}`)

	var subject QC20Token

	if err := json.Unmarshal(
		legacyJSON,
		&subject,
	); err != nil {
		t.Fatalf("load legacy QC20 JSON: %v", err)
	}

	next, err := subject.NextAuthorizationNonceV1(
		owner,
	)
	if err != nil {
		t.Fatalf("legacy next nonce: %v", err)
	}

	if next != 1 {
		t.Fatalf(
			"legacy next nonce = %d, want 1",
			next,
		)
	}

	if balance := subject.BalanceOf(
		owner,
	); balance != 100 {
		t.Fatalf(
			"legacy balance = %d, want 100",
			balance,
		)
	}

	if allowance := subject.Allowance(
		owner,
		spender,
	); allowance != 25 {
		t.Fatalf(
			"legacy allowance = %d, want 25",
			allowance,
		)
	}
}

func TestQC20AuthorizationStateV1DomainIsStable(
	t *testing.T,
) {
	owner := wallet.NewWallet().GetAddress()

	subject := NewQC20TokenWithOwner(
		"Stable Factory Token",
		"SFT",
		1000,
		8,
		owner,
	)

	first, err := subject.AuthorizationDomainV1()
	if err != nil {
		t.Fatalf("first domain: %v", err)
	}

	second, err := subject.AuthorizationDomainV1()
	if err != nil {
		t.Fatalf("second domain: %v", err)
	}

	if first == "" || first != second {
		t.Fatal("QC20 authorization domain is not stable")
	}

	next, err := subject.NextAuthorizationNonceV1(owner)
	if err != nil {
		t.Fatalf("next nonce: %v", err)
	}

	if next != 1 {
		t.Fatalf("new QC20 nonce = %d, want 1", next)
	}
}
