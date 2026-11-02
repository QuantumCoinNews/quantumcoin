package token

import (
	"errors"
	"sync"
	"testing"

	"quantumcoin/security"
	"quantumcoin/wallet"
)

func signedQC721OwnerOperationV1(
	t *testing.T,
	subject *QC721Token,
	signer *wallet.Wallet,
	operation string,
	to string,
	tokenID uint64,
	nonce uint64,
) TokenAuthorizationV1 {
	t.Helper()

	domain, err := subject.AuthorizationDomainV1()
	if err != nil {
		t.Fatalf("QC721 domain: %v", err)
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
			TokenID:   tokenID,
			Nonce:     nonce,
		},
		signer.PrivateKey,
	)
	if err != nil {
		t.Fatalf("sign QC721 operation: %v", err)
	}

	return authorization
}

func newQC721TestTokenV1(
	t *testing.T,
	collectionOwner *wallet.Wallet,
	holder *wallet.Wallet,
	tokenID uint64,
) *QC721Token {
	t.Helper()

	subject := NewQC721TokenWithOwner(
		"Factory NFT",
		"FNFT",
		collectionOwner.GetAddress(),
	)

	if err := subject.MintByOwner(
		collectionOwner.GetAddress(),
		holder.GetAddress(),
		tokenID,
		"ipfs://factory-nft",
	); err != nil {
		t.Fatalf("mint test NFT: %v", err)
	}

	return subject
}

func TestQC721OwnerOperationsV1TransferThenNewOwnerBurns(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	collectionOwner := wallet.NewWallet()
	holder := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := newQC721TestTokenV1(
		t,
		collectionOwner,
		holder,
		7,
	)

	transfer := signedQC721OwnerOperationV1(
		t,
		subject,
		holder,
		TokenOperationTransferV1,
		recipient.GetAddress(),
		7,
		1,
	)

	if err := subject.TransferAuthorizedV1(
		transfer,
		recipient.GetAddress(),
		7,
	); err != nil {
		t.Fatalf("authorized transfer: %v", err)
	}

	owner, err := subject.OwnerOf(7)
	if err != nil || owner != recipient.GetAddress() {
		t.Fatalf("owner=%q err=%v", owner, err)
	}

	burn := signedQC721OwnerOperationV1(
		t,
		subject,
		recipient,
		TokenOperationBurnV1,
		"",
		7,
		1,
	)

	if err := subject.BurnAuthorizedV1(
		burn,
		7,
	); err != nil {
		t.Fatalf("authorized burn: %v", err)
	}

	if _, err := subject.OwnerOf(7); err == nil {
		t.Fatal("burned NFT still has an owner")
	}

	if _, err := subject.TokenURI(7); err == nil {
		t.Fatal("burned NFT still has a URI")
	}

	holderNext, err :=
		subject.NextAuthorizationNonceV1(
			holder.GetAddress(),
		)
	if err != nil || holderNext != 2 {
		t.Fatalf(
			"holder next nonce=%d err=%v",
			holderNext,
			err,
		)
	}

	recipientNext, err :=
		subject.NextAuthorizationNonceV1(
			recipient.GetAddress(),
		)
	if err != nil || recipientNext != 2 {
		t.Fatalf(
			"recipient next nonce=%d err=%v",
			recipientNext,
			err,
		)
	}
}

func TestQC721OwnerOperationsV1LegacyMethodsAreLocked(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	collectionOwner := wallet.NewWallet()
	holder := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := newQC721TestTokenV1(
		t,
		collectionOwner,
		holder,
		7,
	)

	transferErr := subject.Transfer(
		holder.GetAddress(),
		recipient.GetAddress(),
		7,
	)

	burnErr := subject.Burn(
		7,
		holder.GetAddress(),
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

	owner, err := subject.OwnerOf(7)
	if err != nil || owner != holder.GetAddress() {
		t.Fatal("legacy operation changed NFT owner")
	}

	if _, err := subject.TokenURI(7); err != nil {
		t.Fatal("legacy operation removed NFT URI")
	}

	events := security.ReadEvents(directory, 10).Events
	if len(events) != 2 {
		t.Fatalf("legacy event count = %d", len(events))
	}
}

func TestQC721OwnerOperationsV1RejectTamperingAndReplay(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	collectionOwner := wallet.NewWallet()
	holder := wallet.NewWallet()
	recipient := wallet.NewWallet()
	otherRecipient := wallet.NewWallet()

	subject := newQC721TestTokenV1(
		t,
		collectionOwner,
		holder,
		7,
	)

	transfer := signedQC721OwnerOperationV1(
		t,
		subject,
		holder,
		TokenOperationTransferV1,
		recipient.GetAddress(),
		7,
		1,
	)

	err := subject.TransferAuthorizedV1(
		transfer,
		recipient.GetAddress(),
		8,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("tampered tokenID error = %v", err)
	}

	err = subject.TransferAuthorizedV1(
		transfer,
		otherRecipient.GetAddress(),
		7,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("tampered recipient error = %v", err)
	}

	if err := subject.TransferAuthorizedV1(
		transfer,
		recipient.GetAddress(),
		7,
	); err != nil {
		t.Fatalf("valid transfer: %v", err)
	}

	err = subject.TransferAuthorizedV1(
		transfer,
		recipient.GetAddress(),
		7,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationReplay,
	) {
		t.Fatalf("replay error = %v", err)
	}
}

func TestQC721OwnerOperationsV1OwnershipFailureDoesNotConsumeNonce(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	collectionOwner := wallet.NewWallet()
	holder := wallet.NewWallet()
	attacker := wallet.NewWallet()

	subject := newQC721TestTokenV1(
		t,
		collectionOwner,
		holder,
		7,
	)

	attackerBurn := signedQC721OwnerOperationV1(
		t,
		subject,
		attacker,
		TokenOperationBurnV1,
		"",
		7,
		1,
	)

	if err := subject.BurnAuthorizedV1(
		attackerBurn,
		7,
	); err == nil {
		t.Fatal("non-owner burn succeeded")
	}

	next, err := subject.NextAuthorizationNonceV1(
		attacker.GetAddress(),
	)
	if err != nil || next != 1 {
		t.Fatalf(
			"failed ownership check consumed nonce: next=%d err=%v",
			next,
			err,
		)
	}

	holderBurn := signedQC721OwnerOperationV1(
		t,
		subject,
		holder,
		TokenOperationBurnV1,
		"",
		7,
		1,
	)

	if err := subject.BurnAuthorizedV1(
		holderBurn,
		7,
	); err != nil {
		t.Fatalf("owner burn: %v", err)
	}
}

func TestQC721OwnerOperationsV1SerializeSameNonce(
	t *testing.T,
) {
	t.Setenv("QC_NODE_DIR", t.TempDir())

	collectionOwner := wallet.NewWallet()
	holder := wallet.NewWallet()
	firstRecipient := wallet.NewWallet()
	secondRecipient := wallet.NewWallet()

	subject := newQC721TestTokenV1(
		t,
		collectionOwner,
		holder,
		7,
	)

	first := signedQC721OwnerOperationV1(
		t,
		subject,
		holder,
		TokenOperationTransferV1,
		firstRecipient.GetAddress(),
		7,
		1,
	)

	second := signedQC721OwnerOperationV1(
		t,
		subject,
		holder,
		TokenOperationTransferV1,
		secondRecipient.GetAddress(),
		7,
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
			first,
			firstRecipient.GetAddress(),
			7,
		)
	}()

	go func() {
		defer wait.Done()
		<-start
		results <- subject.TransferAuthorizedV1(
			second,
			secondRecipient.GetAddress(),
			7,
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

	owner, err := subject.OwnerOf(7)
	if err != nil {
		t.Fatalf("owner after race: %v", err)
	}

	if owner != firstRecipient.GetAddress() &&
		owner != secondRecipient.GetAddress() {
		t.Fatalf("unexpected NFT owner: %q", owner)
	}

	next, err := subject.NextAuthorizationNonceV1(
		holder.GetAddress(),
	)
	if err != nil || next != 2 {
		t.Fatalf(
			"holder next nonce=%d err=%v",
			next,
			err,
		)
	}
}
