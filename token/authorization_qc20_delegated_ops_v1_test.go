package token

import (
	"errors"
	"sync"
	"testing"

	"quantumcoin/security"
	"quantumcoin/wallet"
)

func signedQC20ApproveV1(
	t *testing.T,
	subject *QC20Token,
	owner *wallet.Wallet,
	spender string,
	amount uint64,
	nonce uint64,
) TokenAuthorizationV1 {
	t.Helper()

	domain, err := subject.AuthorizationDomainV1()
	if err != nil {
		t.Fatalf("QC20 domain: %v", err)
	}

	caller := owner.GetAddress()

	authorization, err := SignTokenAuthorizationV1(
		TokenAuthorizationClaimV1{
			Version:   TokenAuthorizationVersionV1,
			Domain:    domain,
			Operation: TokenOperationApproveV1,
			Caller:    caller,
			From:      caller,
			Spender:   spender,
			Amount:    amount,
			Nonce:     nonce,
		},
		owner.PrivateKey,
	)
	if err != nil {
		t.Fatalf("sign approve: %v", err)
	}

	return authorization
}

func signedQC20TransferFromV1(
	t *testing.T,
	subject *QC20Token,
	spender *wallet.Wallet,
	owner string,
	recipient string,
	amount uint64,
	nonce uint64,
) TokenAuthorizationV1 {
	t.Helper()

	domain, err := subject.AuthorizationDomainV1()
	if err != nil {
		t.Fatalf("QC20 domain: %v", err)
	}

	caller := spender.GetAddress()

	authorization, err := SignTokenAuthorizationV1(
		TokenAuthorizationClaimV1{
			Version:   TokenAuthorizationVersionV1,
			Domain:    domain,
			Operation: TokenOperationTransferFromV1,
			Caller:    caller,
			From:      owner,
			To:        recipient,
			Spender:   caller,
			Amount:    amount,
			Nonce:     nonce,
		},
		spender.PrivateKey,
	)
	if err != nil {
		t.Fatalf("sign transferFrom: %v", err)
	}

	return authorization
}

func TestQC20DelegatedOperationsV1ApproveTransferAndRevoke(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	spender := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Delegated Token",
		"DLG",
		1000,
		8,
		owner.GetAddress(),
	)

	approve := signedQC20ApproveV1(
		t,
		subject,
		owner,
		spender.GetAddress(),
		100,
		1,
	)

	if err := subject.ApproveAuthorizedV1(
		approve,
		spender.GetAddress(),
		100,
	); err != nil {
		t.Fatalf("approve: %v", err)
	}

	transfer := signedQC20TransferFromV1(
		t,
		subject,
		spender,
		owner.GetAddress(),
		recipient.GetAddress(),
		40,
		1,
	)

	if err := subject.TransferFromAuthorizedV1(
		transfer,
		owner.GetAddress(),
		recipient.GetAddress(),
		40,
	); err != nil {
		t.Fatalf("transferFrom: %v", err)
	}

	if allowance := subject.Allowance(
		owner.GetAddress(),
		spender.GetAddress(),
	); allowance != 60 {
		t.Fatalf("allowance = %d, want 60", allowance)
	}

	revoke := signedQC20ApproveV1(
		t,
		subject,
		owner,
		spender.GetAddress(),
		0,
		2,
	)

	if err := subject.ApproveAuthorizedV1(
		revoke,
		spender.GetAddress(),
		0,
	); err != nil {
		t.Fatalf("revoke allowance: %v", err)
	}

	if subject.Allowance(
		owner.GetAddress(),
		spender.GetAddress(),
	) != 0 {
		t.Fatal("allowance was not revoked")
	}

	if subject.BalanceOf(owner.GetAddress()) != 960 ||
		subject.BalanceOf(recipient.GetAddress()) != 40 {
		t.Fatal("delegated transfer balances are incorrect")
	}

	ownerNext, err := subject.NextAuthorizationNonceV1(
		owner.GetAddress(),
	)
	if err != nil || ownerNext != 3 {
		t.Fatalf(
			"owner next nonce=%d err=%v",
			ownerNext,
			err,
		)
	}

	spenderNext, err := subject.NextAuthorizationNonceV1(
		spender.GetAddress(),
	)
	if err != nil || spenderNext != 2 {
		t.Fatalf(
			"spender next nonce=%d err=%v",
			spenderNext,
			err,
		)
	}
}

func TestQC20DelegatedOperationsV1LegacyMethodsAreLocked(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	owner := wallet.NewWallet()
	spender := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Locked Delegation",
		"LDG",
		1000,
		8,
		owner.GetAddress(),
	)

	approveErr := subject.Approve(
		owner.GetAddress(),
		spender.GetAddress(),
		100,
	)

	transferErr := subject.TransferFrom(
		owner.GetAddress(),
		spender.GetAddress(),
		recipient.GetAddress(),
		25,
	)

	if !errors.Is(
		approveErr,
		ErrTokenAuthorizationRequired,
	) {
		t.Fatalf("legacy approve error = %v", approveErr)
	}

	if !errors.Is(
		transferErr,
		ErrTokenAuthorizationRequired,
	) {
		t.Fatalf(
			"legacy transferFrom error = %v",
			transferErr,
		)
	}

	if subject.Allowance(
		owner.GetAddress(),
		spender.GetAddress(),
	) != 0 {
		t.Fatal("legacy approve changed allowance")
	}

	if subject.BalanceOf(owner.GetAddress()) != 1000 ||
		subject.BalanceOf(recipient.GetAddress()) != 0 {
		t.Fatal("legacy transferFrom changed balances")
	}

	events := security.ReadEvents(directory, 10).Events

	if len(events) != 2 {
		t.Fatalf("legacy event count = %d", len(events))
	}
}

func TestQC20DelegatedOperationsV1RejectTamperingAndReplay(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	spender := wallet.NewWallet()
	otherSpender := wallet.NewWallet()
	recipient := wallet.NewWallet()
	otherRecipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Tamper Token",
		"TMP",
		1000,
		8,
		owner.GetAddress(),
	)

	approve := signedQC20ApproveV1(
		t,
		subject,
		owner,
		spender.GetAddress(),
		50,
		1,
	)

	err := subject.ApproveAuthorizedV1(
		approve,
		otherSpender.GetAddress(),
		50,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("tampered spender error = %v", err)
	}

	if err := subject.ApproveAuthorizedV1(
		approve,
		spender.GetAddress(),
		50,
	); err != nil {
		t.Fatalf("valid approve: %v", err)
	}

	err = subject.ApproveAuthorizedV1(
		approve,
		spender.GetAddress(),
		50,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationReplay,
	) {
		t.Fatalf("approve replay error = %v", err)
	}

	transfer := signedQC20TransferFromV1(
		t,
		subject,
		spender,
		owner.GetAddress(),
		recipient.GetAddress(),
		20,
		1,
	)

	err = subject.TransferFromAuthorizedV1(
		transfer,
		owner.GetAddress(),
		otherRecipient.GetAddress(),
		20,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("tampered recipient error = %v", err)
	}

	err = subject.TransferFromAuthorizedV1(
		approve,
		owner.GetAddress(),
		recipient.GetAddress(),
		20,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("operation substitution error = %v", err)
	}

	if err := subject.TransferFromAuthorizedV1(
		transfer,
		owner.GetAddress(),
		recipient.GetAddress(),
		20,
	); err != nil {
		t.Fatalf("valid transferFrom: %v", err)
	}

	err = subject.TransferFromAuthorizedV1(
		transfer,
		owner.GetAddress(),
		recipient.GetAddress(),
		20,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationReplay,
	) {
		t.Fatalf("transferFrom replay error = %v", err)
	}
}

func TestQC20DelegatedOperationsV1FailureDoesNotConsumeNonce(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	spender := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Failure Token",
		"FLR",
		1000,
		8,
		owner.GetAddress(),
	)

	approve := signedQC20ApproveV1(
		t,
		subject,
		owner,
		spender.GetAddress(),
		10,
		1,
	)

	if err := subject.ApproveAuthorizedV1(
		approve,
		spender.GetAddress(),
		10,
	); err != nil {
		t.Fatalf("approve: %v", err)
	}

	failing := signedQC20TransferFromV1(
		t,
		subject,
		spender,
		owner.GetAddress(),
		recipient.GetAddress(),
		20,
		1,
	)

	if err := subject.TransferFromAuthorizedV1(
		failing,
		owner.GetAddress(),
		recipient.GetAddress(),
		20,
	); err == nil {
		t.Fatal("over-allowance transfer succeeded")
	}

	next, err := subject.NextAuthorizationNonceV1(
		spender.GetAddress(),
	)
	if err != nil || next != 1 {
		t.Fatalf(
			"failed transfer consumed nonce: next=%d err=%v",
			next,
			err,
		)
	}

	valid := signedQC20TransferFromV1(
		t,
		subject,
		spender,
		owner.GetAddress(),
		recipient.GetAddress(),
		10,
		1,
	)

	if err := subject.TransferFromAuthorizedV1(
		valid,
		owner.GetAddress(),
		recipient.GetAddress(),
		10,
	); err != nil {
		t.Fatalf("valid transferFrom: %v", err)
	}
}

func TestQC20DelegatedOperationsV1SerializeSameSpenderNonce(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	spender := wallet.NewWallet()
	firstRecipient := wallet.NewWallet()
	secondRecipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Concurrent Delegation",
		"CDG",
		1000,
		8,
		owner.GetAddress(),
	)

	approve := signedQC20ApproveV1(
		t,
		subject,
		owner,
		spender.GetAddress(),
		100,
		1,
	)

	if err := subject.ApproveAuthorizedV1(
		approve,
		spender.GetAddress(),
		100,
	); err != nil {
		t.Fatalf("approve: %v", err)
	}

	first := signedQC20TransferFromV1(
		t,
		subject,
		spender,
		owner.GetAddress(),
		firstRecipient.GetAddress(),
		10,
		1,
	)

	second := signedQC20TransferFromV1(
		t,
		subject,
		spender,
		owner.GetAddress(),
		secondRecipient.GetAddress(),
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
		results <- subject.TransferFromAuthorizedV1(
			first,
			owner.GetAddress(),
			firstRecipient.GetAddress(),
			10,
		)
	}()

	go func() {
		defer wait.Done()
		<-start
		results <- subject.TransferFromAuthorizedV1(
			second,
			owner.GetAddress(),
			secondRecipient.GetAddress(),
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
	firstBalance := subject.BalanceOf(
		firstRecipient.GetAddress(),
	)
	secondBalance := subject.BalanceOf(
		secondRecipient.GetAddress(),
	)

	if ownerBalance+firstBalance+secondBalance != 1000 {
		t.Fatal("delegated transfer changed total balance")
	}

	used := firstBalance + secondBalance
	remaining := subject.Allowance(
		owner.GetAddress(),
		spender.GetAddress(),
	)

	if used+remaining != 100 {
		t.Fatalf(
			"used=%d remaining=%d",
			used,
			remaining,
		)
	}

	next, err := subject.NextAuthorizationNonceV1(
		spender.GetAddress(),
	)
	if err != nil || next != 2 {
		t.Fatalf(
			"spender next nonce=%d err=%v",
			next,
			err,
		)
	}
}
