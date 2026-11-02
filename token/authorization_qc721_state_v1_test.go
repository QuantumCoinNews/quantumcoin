package token

import (
	"encoding/json"
	"errors"
	"testing"

	"quantumcoin/wallet"
)

func TestQC721AuthorizationStateV1PersistsNonceAndNFT(
	t *testing.T,
) {
	collectionOwner := wallet.NewWallet().GetAddress()
	holder := wallet.NewWallet().GetAddress()

	subject := NewQC721TokenWithOwner(
		"Factory NFT",
		"FNFT",
		collectionOwner,
	)

	if err := subject.MintByOwner(
		collectionOwner,
		holder,
		7,
		"ipfs://factory-token-7",
	); err != nil {
		t.Fatalf("mint test NFT: %v", err)
	}

	subject.mu.Lock()
	commitTokenAuthorizationNonceV1(
		subject.authorizationNonces,
		holder,
		1,
	)
	subject.mu.Unlock()

	raw, err := json.Marshal(subject)
	if err != nil {
		t.Fatalf("marshal QC721: %v", err)
	}

	var loaded QC721Token

	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal QC721: %v", err)
	}

	next, err := loaded.NextAuthorizationNonceV1(holder)
	if err != nil {
		t.Fatalf("persisted next nonce: %v", err)
	}

	if next != 2 {
		t.Fatalf("next nonce = %d, want 2", next)
	}

	loaded.mu.Lock()
	replayErr := checkTokenAuthorizationNonceV1(
		loaded.authorizationNonces,
		holder,
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

	actualOwner, err := loaded.OwnerOf(7)
	if err != nil {
		t.Fatalf("loaded owner: %v", err)
	}

	if actualOwner != holder {
		t.Fatalf(
			"loaded owner = %q, want %q",
			actualOwner,
			holder,
		)
	}

	uri, err := loaded.TokenURI(7)
	if err != nil {
		t.Fatalf("loaded token URI: %v", err)
	}

	if uri != "ipfs://factory-token-7" {
		t.Fatalf("loaded URI = %q", uri)
	}
}

func TestQC721AuthorizationStateV1LoadsLegacyJSON(
	t *testing.T,
) {
	collectionOwner := canonicalAddressFixture(
		"collection-owner",
	)
	holder := canonicalAddressFixture(
		"legacy-holder",
	)

	legacyJSON := []byte(`{
"Name":"Legacy NFT",
"Symbol":"LNFT",
"Owner":"` + collectionOwner + `",
"Owners":{"7":"` + holder + `"},
"TokenURIs":{"7":"ipfs://legacy-7"}
}`)

	var subject QC721Token

	if err := json.Unmarshal(
		legacyJSON,
		&subject,
	); err != nil {
		t.Fatalf("load legacy QC721 JSON: %v", err)
	}

	next, err := subject.NextAuthorizationNonceV1(
		holder,
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

	owner, err := subject.OwnerOf(7)
	if err != nil || owner != holder {
		t.Fatalf(
			"legacy owner=%q err=%v",
			owner,
			err,
		)
	}

	uri, err := subject.TokenURI(7)
	if err != nil || uri != "ipfs://legacy-7" {
		t.Fatalf(
			"legacy URI=%q err=%v",
			uri,
			err,
		)
	}
}

func TestQC721AuthorizationStateV1DomainIsStable(
	t *testing.T,
) {
	collectionOwner := wallet.NewWallet().GetAddress()

	subject := NewQC721TokenWithOwner(
		"Stable NFT",
		"SNFT",
		collectionOwner,
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
		t.Fatal("QC721 domain is not stable")
	}

	holder := wallet.NewWallet().GetAddress()

	next, err := subject.NextAuthorizationNonceV1(holder)
	if err != nil {
		t.Fatalf("next nonce: %v", err)
	}

	if next != 1 {
		t.Fatalf("new holder nonce = %d, want 1", next)
	}
}
