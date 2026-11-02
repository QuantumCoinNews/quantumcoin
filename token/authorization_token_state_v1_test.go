package token

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"quantumcoin/wallet"
)

func TestTokenAuthorizationStateV1PersistsNonce(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	owner := wallet.NewWallet().GetAddress()

	subject := New(
		"TST",
		"Test Token",
		8,
		1000,
		owner,
	)

	if err := subject.Mint(
		owner,
		owner,
		100,
	); err != nil {
		t.Fatalf(
			"mint persisted token state: %v", err)
	}

	subject.mu.Lock()
	commitTokenAuthorizationNonceV1(
		subject.authorizationNonces,
		owner,
		1,
	)
	subject.mu.Unlock()
	registry := NewRegistry()

	if err := registry.Register(subject); err != nil {
		t.Fatalf("register token: %v", err)
	}

	if err := registry.Save(directory); err != nil {
		t.Fatalf("save token registry: %v", err)
	}

	raw, err := os.ReadFile(
		filepath.Join(directory, registryFile),
	)
	if err != nil {
		t.Fatalf("read token registry: %v", err)
	}

	if !json.Valid(raw) {
		t.Fatal("saved token registry is not valid JSON")
	}

	loadedRegistry := NewRegistry()

	if err := loadedRegistry.Load(directory); err != nil {
		t.Fatalf("load token registry: %v", err)
	}

	loaded, ok := loadedRegistry.Get("TST")
	if !ok || loaded == nil {
		t.Fatal("persisted token was not loaded")
	}

	next, err := loaded.NextAuthorizationNonceV1(owner)
	if err != nil {
		t.Fatalf("read next persisted nonce: %v", err)
	}

	if next != 2 {
		t.Fatalf(
			"next nonce = %d, want 2",
			next,
		)
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

	if balance := loaded.BalanceOf(owner); balance != 100 {
		t.Fatalf(
			"loaded balance = %d, want 100",
			balance,
		)
	}

	originalDomain, err := subject.AuthorizationDomainV1()
	if err != nil {
		t.Fatalf("original domain: %v", err)
	}

	loadedDomain, err := loaded.AuthorizationDomainV1()
	if err != nil {
		t.Fatalf("loaded domain: %v", err)
	}

	if loadedDomain != originalDomain {
		t.Fatal(
			"authorization domain changed after reload",
		)
	}
}

func TestTokenAuthorizationStateV1LoadsLegacyJSON(
	t *testing.T,
) {
	owner := canonicalAddressFixture(
		"legacy-owner",
	)

	legacyJSON := []byte(`{
"symbol":"OLD",
"name":"Legacy Token",
"decimals":8,
"max_supply":1000,
"total_supply":10,
"owner":"` + owner + `",
"balances":{"` + owner + `":10}
}`)

	var subject Token

	if err := json.Unmarshal(
		legacyJSON,
		&subject,
	); err != nil {
		t.Fatalf("load legacy token JSON: %v", err)
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
	); balance != 10 {
		t.Fatalf(
			"legacy balance = %d, want 10",
			balance,
		)
	}
}

func TestTokenAuthorizationStateV1DomainIsStable(
	t *testing.T,
) {
	owner := wallet.NewWallet().GetAddress()

	subject := New(
		"TST",
		"Stable Domain Token",
		8,
		1000,
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
		t.Fatal(
			"token authorization domain is not stable",
		)
	}

	next, err := subject.NextAuthorizationNonceV1(owner)
	if err != nil {
		t.Fatalf("next nonce: %v", err)
	}

	if next != 1 {
		t.Fatalf(
			"new token nonce = %d, want 1",
			next,
		)
	}
}
