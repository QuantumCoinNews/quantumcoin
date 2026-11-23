package token

import (
	"bytes"
	"errors"
	"strconv"
	"testing"

	"quantumcoin/wallet"
)

func TestQC20ValidatedInitialSupplyBoundariesAndPersistence(t *testing.T) {
	owner := canonicalAddressFixture("initial-supply-owner")
	for _, supply := range []uint64{0, 1, 1_000_000, 1<<53 + 1, ^uint64(0)} {
		t.Run(strconv.FormatUint(supply, 10), func(t *testing.T) {
			subject, err := NewValidatedQC20TokenV1("Fixed Token", "FIX", supply, 8, owner)
			if err != nil {
				t.Fatal(err)
			}
			if subject.Owner() != owner || subject.MaxSupply() != supply || subject.TotalSupply() != supply || subject.BalanceOf(owner) != supply {
				t.Fatal("initial supply was not assigned exactly to the owner")
			}
			balances := subject.BalancesSnapshotV1()
			wantEntries := 1
			if supply == 0 {
				wantEntries = 0
			}
			if len(balances) != wantEntries {
				t.Fatalf("balance entries = %d, want %d", len(balances), wantEntries)
			}
			raw, err := MarshalFungibleTokenV1(subject)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := UnmarshalFungibleTokenV1(raw)
			if err != nil {
				t.Fatal(err)
			}
			after, err := MarshalFungibleTokenV1(loaded)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(raw, after) {
				t.Fatal("persistence changed exact initial state")
			}
		})
	}
}

func TestQC20ValidatedInitialSupplyRejectsInvalidOwners(t *testing.T) {
	owner := canonicalAddressFixture("initial-supply-owner")
	badChecksum := owner[:len(owner)-1] + "1"
	if owner[len(owner)-1] == '1' {
		badChecksum = owner[:len(owner)-1] + "2"
	}
	for name, address := range map[string]string{
		"empty": "", "whitespace": " ", "placeholder": "token-owner",
		"noncanonical": " " + owner, "bad checksum": badChecksum,
		"wrong network": addressForPolicyVersion(TokenAddressNetworkVersionV1^1, 0x75),
	} {
		t.Run(name, func(t *testing.T) {
			for _, supply := range []uint64{0, 100} {
				subject, err := NewValidatedQC20TokenV1("Fixed Token", "FIX", supply, 8, address)
				if subject != nil || !errors.Is(err, ErrInvalidTokenAddress) {
					t.Fatalf("invalid owner produced token or wrong error: %v", err)
				}
			}
		})
	}
}

func TestQC20InitialOwnerLifecycleSurvivesReloadWithoutRemint(t *testing.T) {
	t.Setenv("QC_NODE_DIR", t.TempDir())
	owner, recipient := wallet.NewWallet(), wallet.NewWallet()
	subject, err := NewValidatedQC20TokenForNetworkV1("Fixed Token", "FIX", 100, 8, owner.GetAddress(), testTokenNetworkMagicV1, testTokenGenesisHashV1)
	if err != nil {
		t.Fatal(err)
	}
	domain, err := subject.AuthorizationDomainV1()
	if err != nil {
		t.Fatal(err)
	}
	transfer := signedQC20OwnerOperationV1(t, subject, owner, TokenOperationTransferV1, recipient.GetAddress(), 40, 1)
	if err := subject.TransferAuthorizedV1(transfer, recipient.GetAddress(), 40); err != nil {
		t.Fatal(err)
	}
	raw, err := MarshalFungibleTokenV1(subject)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := UnmarshalFungibleTokenV1(raw)
	if err != nil {
		t.Fatal(err)
	}
	restored, ok := loaded.(*QC20Token)
	if !ok {
		t.Fatal("fixed-supply model changed on reload")
	}
	if err := restored.TransferAuthorizedV1(transfer, recipient.GetAddress(), 40); !errors.Is(err, ErrTokenAuthorizationReplay) {
		t.Fatalf("replayed initial-owner transfer accepted: %v", err)
	}
	burn := signedQC20OwnerOperationV1(t, restored, owner, TokenOperationBurnV1, "", 10, 2)
	if err := restored.BurnAuthorizedV1(burn, 10); err != nil {
		t.Fatal(err)
	}
	if restored.Owner() != owner.GetAddress() || restored.MaxSupply() != 100 || restored.TotalSupply() != 90 || restored.BalanceOf(owner.GetAddress()) != 50 || restored.BalanceOf(recipient.GetAddress()) != 40 {
		t.Fatal("transfer/burn changed ownership or supply accounting")
	}
	before, err := MarshalFungibleTokenV1(restored)
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.MintByOwner(owner.GetAddress(), recipient.GetAddress(), 10); !errors.Is(err, ErrQC20FixedSupplyMintDisabled) {
		t.Fatalf("remint: %v", err)
	}
	if err := restored.Mint(recipient.GetAddress(), 10); !errors.Is(err, ErrQC20FixedSupplyMintDisabled) {
		t.Fatalf("legacy remint: %v", err)
	}
	after, err := MarshalFungibleTokenV1(restored)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("rejected remint changed state")
	}
	loaded, err = UnmarshalFungibleTokenV1(after)
	if err != nil {
		t.Fatal(err)
	}
	final, ok := loaded.(*QC20Token)
	if !ok {
		t.Fatal("fixed-supply model changed after burn")
	}
	finalDomain, err := final.AuthorizationDomainV1()
	if err != nil || finalDomain != domain {
		t.Fatal("owner/supply authorization domain changed")
	}
	if nonce, err := final.NextAuthorizationNonceV1(owner.GetAddress()); err != nil || nonce != 3 {
		t.Fatalf("persisted next nonce = %d, err = %v", nonce, err)
	}
	finalRaw, err := MarshalFungibleTokenV1(final)
	if err != nil || !bytes.Equal(after, finalRaw) {
		t.Fatal("final reload changed burned supply or balances")
	}
}
