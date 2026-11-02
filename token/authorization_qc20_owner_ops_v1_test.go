package token

import (
	"errors"
	"sync"
	"testing"

	"quantumcoin/security"
	"quantumcoin/wallet"
)

func signedQC20OwnerOperationV1(
	t *testing.T,
	subject *QC20Token,
	signer *wallet.Wallet,
	operation string,
	to string,
	amount uint64,
	nonce uint64,
) TokenAuthorizationV1 {
	t.Helper()

	domain, err := subject.AuthorizationDomainV1()
	if err != nil {
		t.Fatalf("QC20 domain: %v", err)
	}

	caller := signer.GetAddress()

	authorization, err := SignTokenAuthorizationV1(
		TokenAuthorizationClaimV1{
			Version:   TokenAuthorizationVersionV1,
			Domain:    domain,
			Operation: operation,
			Caller:    caller,
			From:      caller,
			To:        to,
			Amount:    amount,
			Nonce:     nonce,
		},
		signer.PrivateKey,
	)
	if err != nil {
		t.Fatalf("sign QC20 operation: %v", err)
	}

	return authorization
}

func TestQC20OwnerOperationsV1TransferAndBurn(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Factory Token",
		"FCT",
		1000,
		8,
		owner.GetAddress(),
	)

	transfer := signedQC20OwnerOperationV1(
		t,
		subject,
		owner,
		TokenOperationTransferV1,
		recipient.GetAddress(),
		25,
		1,
	)

	if err := subject.TransferAuthorizedV1(
		transfer,
		recipient.GetAddress(),
		25,
	); err != nil {
		t.Fatalf("authorized transfer: %v", err)
	}

	burn := signedQC20OwnerOperationV1(
		t,
		subject,
		owner,
		TokenOperationBurnV1,
		"",
		100,
		2,
	)

	if err := subject.BurnAuthorizedV1(
		burn,
		100,
	); err != nil {
		t.Fatalf("authorized burn: %v", err)
	}

	if balance := subject.BalanceOf(
		owner.GetAddress(),
	); balance != 875 {
		t.Fatalf("owner balance = %d, want 875", balance)
	}

	if balance := subject.BalanceOf(
		recipient.GetAddress(),
	); balance != 25 {
		t.Fatalf(
			"recipient balance = %d, want 25",
			balance,
		)
	}

	if subject.TotalSupply() != 900 {
		t.Fatalf(
			"total supply = %d, want 900",
			subject.TotalSupply(),
		)
	}

	next, err := subject.NextAuthorizationNonceV1(
		owner.GetAddress(),
	)
	if err != nil || next != 3 {
		t.Fatalf("next nonce=%d err=%v", next, err)
	}
}

func TestQC20OwnerOperationsV1LegacyMethodsAreLocked(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Locked Token",
		"LCK",
		1000,
		8,
		owner.GetAddress(),
	)

	transferErr := subject.Transfer(
		owner.GetAddress(),
		recipient.GetAddress(),
		25,
	)

	burnErr := subject.Burn(
		owner.GetAddress(),
		100,
	)

	if !errors.Is(
		transferErr,
		ErrTokenAuthorizationRequired,
	) {
		t.Fatalf("legacy transfer error = %v", transferErr)
	}

	if !errors.Is(
		burnErr,
		ErrTokenAuthorizationRequired,
	) {
		t.Fatalf("legacy burn error = %v", burnErr)
	}

	if subject.BalanceOf(owner.GetAddress()) != 1000 ||
		subject.BalanceOf(recipient.GetAddress()) != 0 ||
		subject.TotalSupply() != 1000 {
		t.Fatal("legacy operation changed QC20 state")
	}

	events := security.ReadEvents(directory, 10).Events

	if len(events) != 2 {
		t.Fatalf("legacy event count = %d", len(events))
	}

	for _, event := range events {
		if event.Type !=
			TokenAuthorizationRejectedEventV1 {
			t.Fatalf(
				"unexpected legacy event: %#v",
				event,
			)
		}
	}
}

func TestQC20OwnerOperationsV1RejectTamperingAndReplay(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Replay Token",
		"RPL",
		1000,
		8,
		owner.GetAddress(),
	)

	transfer := signedQC20OwnerOperationV1(
		t,
		subject,
		owner,
		TokenOperationTransferV1,
		recipient.GetAddress(),
		25,
		1,
	)

	err := subject.TransferAuthorizedV1(
		transfer,
		recipient.GetAddress(),
		26,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("tampered transfer error = %v", err)
	}

	err = subject.BurnAuthorizedV1(
		transfer,
		25,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("operation substitution error = %v", err)
	}

	if err := subject.TransferAuthorizedV1(
		transfer,
		recipient.GetAddress(),
		25,
	); err != nil {
		t.Fatalf("valid transfer: %v", err)
	}

	err = subject.TransferAuthorizedV1(
		transfer,
		recipient.GetAddress(),
		25,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationReplay,
	) {
		t.Fatalf("replay error = %v", err)
	}

	if subject.BalanceOf(
		recipient.GetAddress(),
	) != 25 {
		t.Fatal("replay changed recipient balance")
	}
}

func TestQC20OwnerOperationsV1FailureDoesNotConsumeNonce(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Failure Token",
		"FLR",
		100,
		8,
		owner.GetAddress(),
	)

	failing := signedQC20OwnerOperationV1(
		t,
		subject,
		owner,
		TokenOperationBurnV1,
		"",
		101,
		1,
	)

	if err := subject.BurnAuthorizedV1(
		failing,
		101,
	); err == nil {
		t.Fatal("over-balance burn succeeded")
	}

	next, err := subject.NextAuthorizationNonceV1(
		owner.GetAddress(),
	)
	if err != nil || next != 1 {
		t.Fatalf(
			"failed burn consumed nonce: next=%d err=%v",
			next,
			err,
		)
	}

	valid := signedQC20OwnerOperationV1(
		t,
		subject,
		owner,
		TokenOperationBurnV1,
		"",
		40,
		1,
	)

	if err := subject.BurnAuthorizedV1(
		valid,
		40,
	); err != nil {
		t.Fatalf("valid burn: %v", err)
	}
}

func TestQC20OwnerOperationsV1SerializeSameNonce(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Concurrent Token",
		"CON",
		100,
		8,
		owner.GetAddress(),
	)

	transfer := signedQC20OwnerOperationV1(
		t,
		subject,
		owner,
		TokenOperationTransferV1,
		recipient.GetAddress(),
		10,
		1,
	)

	burn := signedQC20OwnerOperationV1(
		t,
		subject,
		owner,
		TokenOperationBurnV1,
		"",
		20,
		1,
	)

	start := make(chan struct{})
	results := make(chan error, 2)

	var wait sync.WaitGroup
	wait.Add(2)

	go func() {
		defer wait.Done()
		<-start
		results <- subject.TransferAuthorizedV1(
			transfer,
			recipient.GetAddress(),
			10,
		)
	}()

	go func() {
		defer wait.Done()
		<-start
		results <- subject.BurnAuthorizedV1(
			burn,
			20,
		)
	}()

	close(start)
	wait.Wait()
	close(results)

	successes := 0
	replays := 0

	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(
			err,
			ErrTokenAuthorizationReplay,
		):
			replays++
		default:
			t.Fatalf(
				"unexpected concurrent error: %v",
				err,
			)
		}
	}

	if successes != 1 || replays != 1 {
		t.Fatalf(
			"successes=%d replays=%d",
			successes,
			replays,
		)
	}

	ownerBalance := subject.BalanceOf(
		owner.GetAddress(),
	)
	recipientBalance := subject.BalanceOf(
		recipient.GetAddress(),
	)

	if ownerBalance+recipientBalance !=
		subject.TotalSupply() {
		t.Fatalf(
			"balance sum=%d supply=%d",
			ownerBalance+recipientBalance,
			subject.TotalSupply(),
		)
	}

	next, err := subject.NextAuthorizationNonceV1(
		owner.GetAddress(),
	)
	if err != nil || next != 2 {
		t.Fatalf(
			"next nonce=%d err=%v",
			next,
			err,
		)
	}
}
