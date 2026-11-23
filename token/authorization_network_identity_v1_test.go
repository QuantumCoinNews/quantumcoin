package token

import (
	"encoding/json"
	"errors"
	"testing"

	"quantumcoin/wallet"
)

func TestTokenAuthorizationIdentityV1SeparatesNetworkGenesisAndInstance(t *testing.T) {
	owner := wallet.NewWallet()
	first, err := NewValidatedTokenForNetworkV1("TST", "Test", 8, 100, owner.GetAddress(), testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatal(err)
	}
	otherNetwork, err := NewValidatedTokenForNetworkV1("TST", "Test", 8, 100, owner.GetAddress(), testTokenNetworkMagicV1+1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatal(err)
	}
	otherGenesis, err := NewValidatedTokenForNetworkV1("TST", "Test", 8, 100, owner.GetAddress(), testTokenNetworkMagicV1, "0202020202020202020202020202020202020202020202020202020202020202")
	if err != nil {
		t.Fatal(err)
	}
	otherInstance, err := NewValidatedTokenForNetworkV1("TST", "Test", 8, 100, owner.GetAddress(), testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatal(err)
	}

	firstDomain, err := first.AuthorizationDomainV1()
	if err != nil {
		t.Fatal(err)
	}
	for name, candidate := range map[string]*Token{"network": otherNetwork, "genesis": otherGenesis, "instance": otherInstance} {
		domain, domainErr := candidate.AuthorizationDomainV1()
		if domainErr != nil {
			t.Fatalf("%s domain: %v", name, domainErr)
		}
		if domain == firstDomain {
			t.Errorf("authorization domain did not separate %s", name)
		}
	}

	serialized, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	var restored Token
	if err := json.Unmarshal(serialized, &restored); err != nil {
		t.Fatal(err)
	}
	restoredDomain, err := restored.AuthorizationDomainV1()
	if err != nil {
		t.Fatal(err)
	}
	if restoredDomain != firstDomain {
		t.Fatal("authorization identity changed across persistence")
	}
}

func TestUnscopedLegacyTokenCannotAuthorizeSignedTransfer(t *testing.T) {
	owner := wallet.NewWallet()
	legacy := New("TST", "Legacy", 8, 100, owner.GetAddress())
	if _, err := legacy.AuthorizationDomainV1(); !errors.Is(err, ErrTokenNetworkIdentityRequired) {
		t.Fatalf("unscoped domain error = %v", err)
	}
	if err := legacy.TransferAuthorizedV1(TokenAuthorizationV1{}, owner.GetAddress(), 1); !errors.Is(err, ErrTokenNetworkIdentityRequired) {
		t.Fatalf("unscoped transfer error = %v", err)
	}
}

func TestQC20AndQC721AuthorizationDomainsBindChainIdentity(t *testing.T) {
	owner := wallet.NewWallet().GetAddress()
	qc20, err := NewValidatedQC20TokenForNetworkV1("Fixed", "FIX", 10, 8, owner, testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatal(err)
	}
	qc20OtherNetwork, err := NewValidatedQC20TokenForNetworkV1("Fixed", "FIX", 10, 8, owner, testTokenNetworkMagicV1+1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatal(err)
	}
	qc20Domain, err := qc20.AuthorizationDomainV1()
	if err != nil {
		t.Fatal(err)
	}
	qc20OtherDomain, err := qc20OtherNetwork.AuthorizationDomainV1()
	if err != nil {
		t.Fatal(err)
	}
	if qc20Domain == qc20OtherDomain {
		t.Fatal("QC20 authorization domain crossed networks")
	}

	qc721, err := NewValidatedQC721TokenForNetworkV1("Collection", "NFT", owner, testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatal(err)
	}
	qc721OtherGenesis, err := NewValidatedQC721TokenForNetworkV1("Collection", "NFT", owner, testTokenNetworkMagicV1, "0202020202020202020202020202020202020202020202020202020202020202")
	if err != nil {
		t.Fatal(err)
	}
	qc721Domain, err := qc721.AuthorizationDomainV1()
	if err != nil {
		t.Fatal(err)
	}
	qc721OtherDomain, err := qc721OtherGenesis.AuthorizationDomainV1()
	if err != nil {
		t.Fatal(err)
	}
	if qc721Domain == qc721OtherDomain {
		t.Fatal("QC721 authorization domain crossed genesis identities")
	}
}

func TestInitialIssuanceAndPostCreationMintShutdownV1(t *testing.T) {
	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()
	subject, err := NewValidatedTokenWithInitialSupplyForNetworkV1("ISS", "Initial Supply", 8, 100, 40, owner.GetAddress(), testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatal(err)
	}
	if subject.TotalSupply() != 40 || subject.BalanceOf(owner.GetAddress()) != 40 {
		t.Fatal("initial issuance was not assigned atomically")
	}
	if err := subject.Mint(owner.GetAddress(), recipient.GetAddress(), 1); !errors.Is(err, ErrTokenMintDisabled) {
		t.Fatalf("owner post-creation mint = %v", err)
	}
	if err := subject.Mint(owner.GetAddress(), recipient.GetAddress(), 1); !errors.Is(err, ErrTokenMintDisabled) {
		t.Fatalf("spoofed owner mint = %v", err)
	}
	if subject.TotalSupply() != 40 || subject.BalanceOf(recipient.GetAddress()) != 0 {
		t.Fatal("rejected mint changed state")
	}

	collection, err := NewValidatedQC721TokenWithInitialItemsForNetworkV1("Initial Collection", "INIT", owner.GetAddress(), testTokenNetworkMagicV1, testTokenGenesisHashV1, []QC721InitialItemV1{{TokenID: 1, Owner: owner.GetAddress(), TokenURI: "ipfs://initial/1"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := collection.OwnerOf(1); err != nil || got != owner.GetAddress() {
		t.Fatalf("initial NFT owner=%q err=%v", got, err)
	}
	if err := collection.MintByOwner(owner.GetAddress(), recipient.GetAddress(), 2, "ipfs://2"); !errors.Is(err, ErrQC721MintDisabled) {
		t.Fatalf("QC721 post-creation mint = %v", err)
	}
	if err := collection.Mint(recipient.GetAddress(), 3, "ipfs://3"); !errors.Is(err, ErrQC721MintDisabled) {
		t.Fatalf("QC721 legacy mint = %v", err)
	}
	if _, err := collection.OwnerOf(2); err == nil {
		t.Fatal("rejected QC721 mint created an item")
	}
}
