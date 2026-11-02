package token

import (
	"errors"
	"testing"

	"quantumcoin/wallet"
)

func newSignedTokenAuthorizationV1(
	t *testing.T,
	claim TokenAuthorizationClaimV1,
) TokenAuthorizationV1 {
	t.Helper()
	claim.From = canonicalAddressFixture(
		claim.From,
	)
	claim.To = canonicalAddressFixture(
		claim.To,
	)
	claim.Spender = canonicalAddressFixture(
		claim.Spender,
	)
	signer := wallet.NewWallet()

	if claim.Caller == "" {
		claim.Caller = signer.GetAddress()
	}

	authorization, err := SignTokenAuthorizationV1(
		claim,
		signer.PrivateKey,
	)
	if err != nil {
		t.Fatalf(
			"sign token authorization: %v",
			err,
		)
	}

	return authorization
}

func TestTokenAuthorizationV1AcceptsValidSignature(
	t *testing.T,
) {
	signer := wallet.NewWallet()
	caller := signer.GetAddress()

	claim := TokenAuthorizationClaimV1{
		Version: TokenAuthorizationVersionV1,
		Domain: TokenAuthorizationDomainV1(
			TokenAuthorizationModelQC20V1,
			"Test Token",
			"TST",
			caller,
			8,
			1000,
		),
		Operation: TokenOperationTransferV1,
		Caller:    caller,
		From:      caller,
		To:        canonicalAddressFixture("recipient"),
		Amount:    25,
		Nonce:     1,
	}

	authorization, err := SignTokenAuthorizationV1(
		claim,
		signer.PrivateKey,
	)
	if err != nil {
		t.Fatalf("sign authorization: %v", err)
	}

	if err := verifyTokenAuthorizationV1(
		authorization,
		claim,
	); err != nil {
		t.Fatalf(
			"valid authorization rejected: %v",
			err,
		)
	}
}

func TestTokenAuthorizationV1BindsEveryOperationField(
	t *testing.T,
) {
	authorization := newSignedTokenAuthorizationV1(
		t,
		TokenAuthorizationClaimV1{
			Version:   TokenAuthorizationVersionV1,
			Domain:    "field-binding-domain",
			Operation: TokenOperationTransferV1,
			From:      "signed-from",
			To:        "signed-to",
			Amount:    25,
			Nonce:     1,
		},
	)

	expected := authorization.Claim

	testCases := []struct {
		name   string
		change func(*TokenAuthorizationClaimV1)
	}{
		{
			name: "version",
			change: func(c *TokenAuthorizationClaimV1) {
				c.Version = "OTHER_VERSION"
			},
		},
		{
			name: "domain",
			change: func(c *TokenAuthorizationClaimV1) {
				c.Domain += "-other"
			},
		},
		{
			name: "operation",
			change: func(c *TokenAuthorizationClaimV1) {
				c.Operation = TokenOperationBurnV1
			},
		},
		{
			name: "caller",
			change: func(c *TokenAuthorizationClaimV1) {
				c.Caller += "-other"
			},
		},
		{
			name: "from",
			change: func(c *TokenAuthorizationClaimV1) {
				c.From += "-other"
			},
		},
		{
			name: "to",
			change: func(c *TokenAuthorizationClaimV1) {
				c.To += "-other"
			},
		},
		{
			name: "spender",
			change: func(c *TokenAuthorizationClaimV1) {
				c.Spender = "other-spender"
			},
		},
		{
			name: "amount",
			change: func(c *TokenAuthorizationClaimV1) {
				c.Amount++
			},
		},
		{
			name: "tokenID",
			change: func(c *TokenAuthorizationClaimV1) {
				c.TokenID++
			},
		},
		{
			name: "nonce",
			change: func(c *TokenAuthorizationClaimV1) {
				c.Nonce++
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			changed := expected
			testCase.change(&changed)

			err := verifyTokenAuthorizationV1(
				authorization,
				changed,
			)

			if !errors.Is(
				err,
				ErrTokenAuthorizationInvalid,
			) {
				t.Fatalf(
					"changed claim error = %v",
					err,
				)
			}
		})
	}
}

func TestTokenAuthorizationV1RejectsWrongPublicKey(
	t *testing.T,
) {
	authorization := newSignedTokenAuthorizationV1(
		t,
		TokenAuthorizationClaimV1{
			Version:   TokenAuthorizationVersionV1,
			Domain:    "wrong-key-domain",
			Operation: TokenOperationApproveV1,
			Spender:   canonicalAddressFixture("spender"),
			Amount:    10,
			Nonce:     1,
		},
	)

	other := wallet.NewWallet()

	authorization.PublicKey = append(
		[]byte(nil),
		other.PublicKey...,
	)

	err := verifyTokenAuthorizationV1(
		authorization,
		authorization.Claim,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("wrong-key error = %v", err)
	}
}

func TestTokenAuthorizationV1RejectsTamperedSignature(
	t *testing.T,
) {
	authorization := newSignedTokenAuthorizationV1(
		t,
		TokenAuthorizationClaimV1{
			Version:   TokenAuthorizationVersionV1,
			Domain:    "signature-domain",
			Operation: TokenOperationBurnV1,
			From:      "burn-owner",
			Amount:    5,
			Nonce:     1,
		},
	)

	if len(authorization.Signature) == 0 {
		t.Fatal("test signature is empty")
	}

	authorization.Signature[0] ^= 0x01

	err := verifyTokenAuthorizationV1(
		authorization,
		authorization.Claim,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf(
			"tampered-signature error = %v",
			err,
		)
	}
}

func TestTokenAuthorizationV1RejectsCallerMismatch(
	t *testing.T,
) {
	signer := wallet.NewWallet()

	_, err := SignTokenAuthorizationV1(
		TokenAuthorizationClaimV1{
			Version:   TokenAuthorizationVersionV1,
			Domain:    "caller-domain",
			Operation: TokenOperationTransferV1,
			Caller:    "forged-caller",
			From:      "forged-caller",
			To:        canonicalAddressFixture("recipient"),
			Amount:    1,
			Nonce:     1,
		},
		signer.PrivateKey,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationInvalid,
	) {
		t.Fatalf("caller-mismatch error = %v", err)
	}
}

func TestTokenAuthorizationV1RejectsZeroNonce(
	t *testing.T,
) {
	signer := wallet.NewWallet()

	_, err := SignTokenAuthorizationV1(
		TokenAuthorizationClaimV1{
			Version:   TokenAuthorizationVersionV1,
			Domain:    "zero-nonce-domain",
			Operation: TokenOperationTransferV1,
			Caller:    signer.GetAddress(),
			From:      signer.GetAddress(),
			To:        canonicalAddressFixture("recipient"),
			Amount:    1,
			Nonce:     0,
		},
		signer.PrivateKey,
	)

	if !errors.Is(
		err,
		ErrTokenAuthorizationNonce,
	) {
		t.Fatalf("zero-nonce error = %v", err)
	}
}

func TestTokenAuthorizationNonceV1RejectsReplayAndGap(
	t *testing.T,
) {
	nonces := map[string]uint64{}
	caller := "caller"

	if err := checkTokenAuthorizationNonceV1(
		nonces,
		caller,
		1,
	); err != nil {
		t.Fatalf("first nonce rejected: %v", err)
	}

	commitTokenAuthorizationNonceV1(
		nonces,
		caller,
		1,
	)

	if err := checkTokenAuthorizationNonceV1(
		nonces,
		caller,
		1,
	); !errors.Is(err, ErrTokenAuthorizationReplay) {
		t.Fatalf("replay error = %v", err)
	}

	if err := checkTokenAuthorizationNonceV1(
		nonces,
		caller,
		3,
	); !errors.Is(err, ErrTokenAuthorizationNonce) {
		t.Fatalf("nonce-gap error = %v", err)
	}

	if err := checkTokenAuthorizationNonceV1(
		nonces,
		caller,
		2,
	); err != nil {
		t.Fatalf("next nonce rejected: %v", err)
	}
}

func TestTokenAuthorizationDomainV1SeparatesModels(
	t *testing.T,
) {
	legacy := TokenAuthorizationDomainV1(
		TokenAuthorizationModelLegacyV1,
		"Token",
		"TST",
		"owner",
		8,
		1000,
	)

	qc20 := TokenAuthorizationDomainV1(
		TokenAuthorizationModelQC20V1,
		"Token",
		"TST",
		"owner",
		8,
		1000,
	)

	qc721 := TokenAuthorizationDomainV1(
		TokenAuthorizationModelQC721V1,
		"Token",
		"TST",
		"owner",
		8,
		1000,
	)

	if legacy == qc20 ||
		legacy == qc721 ||
		qc20 == qc721 {
		t.Fatal("token model domains are not separated")
	}

	repeated := TokenAuthorizationDomainV1(
		TokenAuthorizationModelQC20V1,
		"Token",
		"TST",
		"owner",
		8,
		1000,
	)

	if repeated != qc20 {
		t.Fatal("authorization domain is not deterministic")
	}
}
