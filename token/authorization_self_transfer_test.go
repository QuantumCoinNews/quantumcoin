package token

import (
	"errors"
	"testing"

	"quantumcoin/wallet"
)

func TestAuthorizedSelfTransferConservesLegacyTokenSupply(t *testing.T) {
	t.Setenv("QC_NODE_DIR", t.TempDir())
	owner := wallet.NewWallet()
	subject := newTestAuthorizedToken(t, "SELF", "Self transfer", 8, 100, owner.GetAddress())
	authorization := signedLegacyTokenTransferV1(t, subject, owner, owner.GetAddress(), 7, 1)
	if err := subject.TransferAuthorizedV1(authorization, owner.GetAddress(), 7); err != nil {
		t.Fatal(err)
	}
	if subject.BalanceOf(owner.GetAddress()) != 100 || subject.GetTotalSupply() != 100 {
		t.Fatalf("self transfer created balance: balance=%d supply=%d", subject.BalanceOf(owner.GetAddress()), subject.GetTotalSupply())
	}
	if err := subject.TransferAuthorizedV1(authorization, owner.GetAddress(), 7); !errors.Is(err, ErrTokenAuthorizationReplay) {
		t.Fatal("self transfer did not consume nonce", err)
	}
}

func TestAuthorizedSelfTransferConservesQC20SupplyAtBalanceBoundary(t *testing.T) {
	for _, supply := range []uint64{100, ^uint64(0)} {
		t.Run(map[bool]string{true: "maximum_balance", false: "normal_balance"}[supply == ^uint64(0)], func(t *testing.T) {
			t.Setenv("QC_NODE_DIR", t.TempDir())
			owner := wallet.NewWallet()
			subject := newTestAuthorizedQC20Token(t, "Self transfer", "SELF", supply, 8, owner.GetAddress())
			authorization := signedQC20OwnerOperationV1(t, subject, owner, TokenOperationTransferV1, owner.GetAddress(), 1, 1)
			if err := subject.TransferAuthorizedV1(authorization, owner.GetAddress(), 1); err != nil {
				t.Fatal("valid self transfer failed", err)
			}
			if subject.BalanceOf(owner.GetAddress()) != supply || subject.TotalSupply() != supply {
				t.Fatalf("self transfer changed supply: balance=%d supply=%d", subject.BalanceOf(owner.GetAddress()), subject.TotalSupply())
			}
			if err := subject.TransferAuthorizedV1(authorization, owner.GetAddress(), 1); !errors.Is(err, ErrTokenAuthorizationReplay) {
				t.Fatal("self transfer did not consume nonce", err)
			}
		})
	}
}

func TestAuthorizedDelegatedSelfTransferConsumesAllowanceWithoutCreatingBalance(t *testing.T) {
	t.Setenv("QC_NODE_DIR", t.TempDir())
	owner, spender := wallet.NewWallet(), wallet.NewWallet()
	subject := newTestAuthorizedQC20Token(t, "Delegated self transfer", "SELF", 100, 8, owner.GetAddress())
	approval := signedQC20ApproveV1(t, subject, owner, spender.GetAddress(), 20, 1)
	if err := subject.ApproveAuthorizedV1(approval, spender.GetAddress(), 20); err != nil {
		t.Fatal(err)
	}
	authorization := signedQC20TransferFromV1(t, subject, spender, owner.GetAddress(), owner.GetAddress(), 5, 1)
	if err := subject.TransferFromAuthorizedV1(authorization, owner.GetAddress(), owner.GetAddress(), 5); err != nil {
		t.Fatal(err)
	}
	if subject.BalanceOf(owner.GetAddress()) != 100 || subject.TotalSupply() != 100 || subject.Allowance(owner.GetAddress(), spender.GetAddress()) != 15 {
		t.Fatal("delegated self transfer inflated balance or bypassed allowance", subject.BalanceOf(owner.GetAddress()), subject.Allowance(owner.GetAddress(), spender.GetAddress()))
	}
	if err := subject.TransferFromAuthorizedV1(authorization, owner.GetAddress(), owner.GetAddress(), 5); !errors.Is(err, ErrTokenAuthorizationReplay) {
		t.Fatal("delegated self transfer did not consume nonce", err)
	}
}
