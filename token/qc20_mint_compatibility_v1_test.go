package token

import (
	"errors"
	"testing"

	"quantumcoin/security"
	"quantumcoin/wallet"
)

func requireQC20MintRejectedAIEventV1(
	t *testing.T,
	directory string,
	caller string,
	expectedErrorClass string,
) {
	t.Helper()

	eventLog := security.ReadEvents(
		directory,
		20,
	)

	if len(eventLog.Events) == 0 {
		t.Fatal("QC20 mint rejection produced no security event")
	}

	event := eventLog.Events[len(eventLog.Events)-1]

	if event.Type != TokenOperationRejectedEventV1 {
		t.Fatalf(
			"mint event type = %q, want %q",
			event.Type,
			TokenOperationRejectedEventV1,
		)
	}

	if event.Severity != security.SeverityWarning {
		t.Fatalf(
			"mint event severity = %q, want %q",
			event.Severity,
			security.SeverityWarning,
		)
	}

	if event.Source != "token" {
		t.Fatalf(
			"mint event source = %q, want token",
			event.Source,
		)
	}

	if event.Subject != caller {
		t.Fatalf(
			"mint event subject = %q, want %q",
			event.Subject,
			caller,
		)
	}

	if event.Metadata["model"] !=
		TokenAuthorizationModelQC20V1 {
		t.Fatalf(
			"mint event model = %#v",
			event.Metadata["model"],
		)
	}

	if event.Metadata["operation"] !=
		TokenOperationMintV1 {
		t.Fatalf(
			"mint event operation = %#v",
			event.Metadata["operation"],
		)
	}

	if event.Metadata["stage"] !=
		tokenAuthorizationStageOperationV1 {
		t.Fatalf(
			"mint event stage = %#v",
			event.Metadata["stage"],
		)
	}

	if event.Metadata["errorClass"] !=
		expectedErrorClass {
		t.Fatalf(
			"mint event errorClass = %#v, want %q",
			event.Metadata["errorClass"],
			expectedErrorClass,
		)
	}

	if event.Metadata["aiReadOnly"] != true {
		t.Fatal("mint event is not AI read-only")
	}

	if event.Metadata["advisoryOnly"] != true {
		t.Fatal("mint event is not advisory-only")
	}

	if event.Metadata["authorizesOperation"] != false {
		t.Fatal("mint event authorizes an operation")
	}

	domainHash, ok :=
		event.Metadata["domainSha256"].(string)

	if !ok || domainHash == "" {
		t.Fatal("mint event has no domain hash")
	}
}

func TestQC20LegacyMintV1ReturnsOwnerConfigurationError(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	subject := NewQC20Token(
		"Ownerless Token",
		"OLT",
		100,
		8,
	)

	err := subject.Mint(
		"recipient",
		1,
	)

	if !errors.Is(
		err,
		ErrTokenOwnerNotConfigured,
	) {
		t.Fatalf(
			"ownerless legacy mint error = %v, want %v",
			err,
			ErrTokenOwnerNotConfigured,
		)
	}

	if subject.TotalSupply() != 0 ||
		subject.BalanceOf("recipient") != 0 {
		t.Fatal("failed ownerless mint changed QC20 state")
	}

	requireQC20MintRejectedAIEventV1(
		t,
		directory,
		"",
		"TOKEN_OPERATION_ERROR",
	)
}

func TestQC20LegacyMintV1ReturnsFixedSupplyError(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	owner := canonicalAddressFixture(
		"owner",
	)
	recipient := canonicalAddressFixture(
		"recipient",
	)

	subject := NewQC20TokenWithOwner(
		"Fixed Token",
		"FIX",
		100,
		8,
		owner,
	)

	err := subject.Mint(
		recipient,
		1,
	)

	if !errors.Is(
		err,
		ErrQC20FixedSupplyMintDisabled,
	) {
		t.Fatalf(
			"legacy mint error = %v, want %v",
			err,
			ErrQC20FixedSupplyMintDisabled,
		)
	}

	if subject.TotalSupply() != 100 ||
		subject.BalanceOf(owner) != 100 ||
		subject.BalanceOf(recipient) != 0 {
		t.Fatal("rejected legacy mint changed QC20 state")
	}

	requireQC20MintRejectedAIEventV1(
		t,
		directory,
		owner,
		"QC20_FIXED_SUPPLY_MINT_DISABLED",
	)
}

func TestQC20LegacyMintV1NilReceiverReturnsError(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	var subject *QC20Token

	if err := subject.Mint(
		"recipient",
		1,
	); err == nil {
		t.Fatal("nil QC20 legacy mint returned no error")
	}

	requireQC20MintRejectedAIEventV1(
		t,
		directory,
		"",
		"TOKEN_OPERATION_ERROR",
	)
}

func TestQC20FixedSupplyV1CannotRemintBurnedSupply(
	t *testing.T,
) {
	directory := t.TempDir()
	t.Setenv("QC_NODE_DIR", directory)

	owner := wallet.NewWallet()
	recipient := wallet.NewWallet()

	subject := NewQC20TokenWithOwner(
		"Permanent Burn Token",
		"PBT",
		100,
		8,
		owner.GetAddress(),
	)

	burn := signedQC20OwnerOperationV1(
		t,
		subject,
		owner,
		TokenOperationBurnV1,
		"",
		40,
		1,
	)

	if err := subject.BurnAuthorizedV1(
		burn,
		40,
	); err != nil {
		t.Fatalf("authorized burn: %v", err)
	}

	if subject.TotalSupply() != 60 ||
		subject.BalanceOf(owner.GetAddress()) != 60 {
		t.Fatal("authorized burn produced unexpected state")
	}

	legacyErr := subject.Mint(
		recipient.GetAddress(),
		40,
	)

	if !errors.Is(
		legacyErr,
		ErrQC20FixedSupplyMintDisabled,
	) {
		t.Fatalf(
			"post-burn legacy mint error = %v, want %v",
			legacyErr,
			ErrQC20FixedSupplyMintDisabled,
		)
	}

	requireQC20MintRejectedAIEventV1(
		t,
		directory,
		owner.GetAddress(),
		"QC20_FIXED_SUPPLY_MINT_DISABLED",
	)

	ownerErr := subject.MintByOwner(
		owner.GetAddress(),
		recipient.GetAddress(),
		40,
	)

	if !errors.Is(
		ownerErr,
		ErrQC20FixedSupplyMintDisabled,
	) {
		t.Fatalf(
			"post-burn owner mint error = %v, want %v",
			ownerErr,
			ErrQC20FixedSupplyMintDisabled,
		)
	}

	requireQC20MintRejectedAIEventV1(
		t,
		directory,
		owner.GetAddress(),
		"QC20_FIXED_SUPPLY_MINT_DISABLED",
	)

	if subject.TotalSupply() != 60 {
		t.Fatalf(
			"rejected remint changed total supply to %d",
			subject.TotalSupply(),
		)
	}

	if subject.BalanceOf(owner.GetAddress()) != 60 {
		t.Fatal("rejected remint changed owner balance")
	}

	if subject.BalanceOf(recipient.GetAddress()) != 0 {
		t.Fatal("rejected remint created recipient balance")
	}

	nextNonce, err := subject.NextAuthorizationNonceV1(
		owner.GetAddress(),
	)
	if err != nil {
		t.Fatalf("next authorization nonce: %v", err)
	}

	if nextNonce != 2 {
		t.Fatalf(
			"next authorization nonce = %d, want 2",
			nextNonce,
		)
	}
}
