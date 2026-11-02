package token

import (
	"errors"
	"sync"
	"testing"

	"quantumcoin/security"
	"quantumcoin/wallet"
)

func signedLegacyTokenTransferV1(
	t *testing.T,
	subject *Token,
	signer *wallet.Wallet,
	to string,
	amount uint64,
	nonce uint64,
) TokenAuthorizationV1 {
	t.Helper()

	domain, err := subject.AuthorizationDomainV1()
	if err != nil {
		t.Fatalf("authorization domain: %v", err)
	}

	caller := signer.GetAddress()

	authorization, err := SignTokenAuthorizationV1(
		TokenAuthorizationClaimV1{
			Version:   TokenAuthorizationVersionV1,
			Domain:    domain,
			Operation: TokenOperationTransferV1,
			Caller:    caller,
			From:      caller,
			To:        to,
			Amount:    amount,
			Nonce:     nonce,
		},
		signer.PrivateKey,
	)
	if err != nil {
		t.Fatalf("sign transfer: %v", err)
	}

	return authorization
}

func TestTokenAuthorizedTransferV1MovesBalanceAndConsumesNonce(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := New(
		"TST",
		"Authorized Token",
		8,
		1000,
		owner.GetAddress(),
	)

	if err := subject.Mint(
		owner.GetAddress(),
		owner.GetAddress(),
		100,
	); err != nil {
		t.Fatalf("mint test balance: %v", err)
	}

	authorization := signedLegacyTokenTransferV1(
		t,
		subject,
		owner,
		recipient.GetAddress(),
		25,
		1,
	)

	if err := subject.TransferAuthorizedV1(
		authorization,
		recipient.GetAddress(),
		25,
	); err != nil {
		t.Fatalf("authorized transfer: %v", err)
	}

	if balance := subject.BalanceOf(
		owner.GetAddress(),
	); balance != 75 {
		t.Fatalf("owner balance = %d, want 75", balance)
	}

	if balance := subject.BalanceOf(
		recipient.GetAddress(),
	); balance != 25 {
		t.Fatalf(
			"recipient balance = %d, want 25",
			balance,
		)
	}

	next, err := subject.NextAuthorizationNonceV1(
		owner.GetAddress(),
	)
	if err != nil {
		t.Fatalf("next nonce: %v", err)
	}

	if next != 2 {
		t.Fatalf("next nonce = %d, want 2", next)
	}

	events := security.ReadEvents(directory, 10).Events

	if len(events) != 1 ||
		events[0].Type !=
			TokenAuthorizationAcceptedEventV1 {
		t.Fatalf(
			"accepted event missing: %#v",
			events,
		)
	}
}

func TestTokenLegacyTransferV1IsLocked(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := New(
		"TST",
		"Locked Legacy Token",
		8,
		1000,
		owner.GetAddress(),
	)

	if err := subject.Mint(
		owner.GetAddress(),
		owner.GetAddress(),
		100,
	); err != nil {
		t.Fatalf("mint test balance: %v", err)
	}

	err := subject.Transfer(
		owner.GetAddress(),
		recipient.GetAddress(),
		25,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationRequired,
	) {
		t.Fatalf("legacy transfer error = %v", err)
	}

	if balance := subject.BalanceOf(
		owner.GetAddress(),
	); balance != 100 {
		t.Fatalf(
			"legacy transfer changed owner balance: %d",
			balance,
		)
	}

	if balance := subject.BalanceOf(
		recipient.GetAddress(),
	); balance != 0 {
		t.Fatalf(
			"legacy transfer changed recipient balance: %d",
			balance,
		)
	}

	events := security.ReadEvents(directory, 10).Events

	if len(events) != 1 ||
		events[0].Type !=
			TokenAuthorizationRejectedEventV1 {
		t.Fatalf(
			"legacy rejection event missing: %#v",
			events,
		)
	}
}

func TestTokenAuthorizedTransferV1RejectsTamperingAndReplay(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := New(
		"TST",
		"Replay Safe Token",
		8,
		1000,
		owner.GetAddress(),
	)

	if err := subject.Mint(
		owner.GetAddress(),
		owner.GetAddress(),
		100,
	); err != nil {
		t.Fatalf("mint test balance: %v", err)
	}

	authorization := signedLegacyTokenTransferV1(
		t,
		subject,
		owner,
		recipient.GetAddress(),
		10,
		1,
	)

	err := subject.TransferAuthorizedV1(
		authorization,
		recipient.GetAddress(),
		11,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("tampered transfer error = %v", err)
	}

	next, err := subject.NextAuthorizationNonceV1(
		owner.GetAddress(),
	)
	if err != nil || next != 1 {
		t.Fatalf(
			"tampered transfer consumed nonce: next=%d err=%v",
			next,
			err,
		)
	}

	if err := subject.TransferAuthorizedV1(
		authorization,
		recipient.GetAddress(),
		10,
	); err != nil {
		t.Fatalf("valid transfer: %v", err)
	}

	err = subject.TransferAuthorizedV1(
		authorization,
		recipient.GetAddress(),
		10,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationReplay,
	) {
		t.Fatalf("replay error = %v", err)
	}

	if balance := subject.BalanceOf(
		recipient.GetAddress(),
	); balance != 10 {
		t.Fatalf(
			"replay changed recipient balance: %d",
			balance,
		)
	}
}

func TestTokenAuthorizedTransferV1DoesNotConsumeNonceOnOperationFailure(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := New(
		"TST",
		"Failure Safe Token",
		8,
		1000,
		owner.GetAddress(),
	)

	if err := subject.Mint(
		owner.GetAddress(),
		owner.GetAddress(),
		100,
	); err != nil {
		t.Fatalf("mint test balance: %v", err)
	}

	failing := signedLegacyTokenTransferV1(
		t,
		subject,
		owner,
		recipient.GetAddress(),
		101,
		1,
	)

	if err := subject.TransferAuthorizedV1(
		failing,
		recipient.GetAddress(),
		101,
	); err == nil {
		t.Fatal("insufficient-balance transfer succeeded")
	}

	next, err := subject.NextAuthorizationNonceV1(
		owner.GetAddress(),
	)
	if err != nil || next != 1 {
		t.Fatalf(
			"failed operation consumed nonce: next=%d err=%v",
			next,
			err,
		)
	}

	valid := signedLegacyTokenTransferV1(
		t,
		subject,
		owner,
		recipient.GetAddress(),
		40,
		1,
	)

	if err := subject.TransferAuthorizedV1(
		valid,
		recipient.GetAddress(),
		40,
	); err != nil {
		t.Fatalf(
			"valid reuse of nonce failed: %v",
			err,
		)
	}
}

func TestTokenAuthorizedTransferV1SerializesSameNonce(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	owner := wallet.NewWallet()
	firstRecipient := wallet.NewWallet()
	secondRecipient := wallet.NewWallet()

	subject := New(
		"TST",
		"Concurrent Token",
		8,
		1000,
		owner.GetAddress(),
	)

	if err := subject.Mint(
		owner.GetAddress(),
		owner.GetAddress(),
		100,
	); err != nil {
		t.Fatalf("mint test balance: %v", err)
	}

	firstAuthorization := signedLegacyTokenTransferV1(
		t,
		subject,
		owner,
		firstRecipient.GetAddress(),
		10,
		1,
	)

	secondAuthorization := signedLegacyTokenTransferV1(
		t,
		subject,
		owner,
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
		results <- subject.TransferAuthorizedV1(
			firstAuthorization,
			firstRecipient.GetAddress(),
			10,
		)
	}()

	go func() {
		defer wait.Done()
		<-start
		results <- subject.TransferAuthorizedV1(
			secondAuthorization,
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
			"successes=%d replays=%d, want 1 and 1",
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

	if ownerBalance+firstBalance+secondBalance != 100 {
		t.Fatalf(
			"concurrent transfer changed total balance",
		)
	}

	next, err := subject.NextAuthorizationNonceV1(
		owner.GetAddress(),
	)
	if err != nil || next != 2 {
		t.Fatalf(
			"concurrent next nonce=%d err=%v",
			next,
			err,
		)
	}
}
