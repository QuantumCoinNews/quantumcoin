package token

import (
	"encoding/json"
	"errors"
	"testing"

	"quantumcoin/utils"
)

func addressForPolicyVersion(
	version byte,
	seed byte,
) string {
	payload := make(
		[]byte,
		tokenAddressPayloadSizeV1,
	)
	payload[0] = version

	for index := 1; index < len(payload); index++ {
		payload[index] = seed
	}

	full := append(
		append([]byte(nil), payload...),
		utils.Checksum(payload)...,
	)

	return string(utils.Base58Encode(full))
}

func corruptAddressChecksum(
	address string,
) string {
	if address == "" {
		return "1"
	}

	replacement := byte('1')
	if address[len(address)-1] == replacement {
		replacement = '2'
	}

	return address[:len(address)-1] +
		string(replacement)
}

func TestTokenAddressPolicyV1AcceptsCanonicalCurrentNetwork(
	t *testing.T,
) {
	address := addressForPolicyVersion(
		TokenAddressNetworkVersionV1,
		0x11,
	)

	canonical, err := NormalizeTokenAddressV1(address)
	if err != nil {
		t.Fatalf("canonical address rejected: %v", err)
	}
	if canonical != address {
		t.Fatalf(
			"canonical address changed: got %q, want %q",
			canonical,
			address,
		)
	}
	if !validTokenAddress(address) {
		t.Fatal("canonical address rejected by token boundary")
	}
}

func TestTokenAddressPolicyV1RejectsInvalidDimensions(
	t *testing.T,
) {
	valid := addressForPolicyVersion(
		TokenAddressNetworkVersionV1,
		0x22,
	)

	tests := []struct {
		name     string
		address  string
		sentinel error
	}{
		{
			name:     "empty",
			address:  "",
			sentinel: ErrTokenAddressFormatV1,
		},
		{
			name:     "whitespace",
			address:  " \t\n ",
			sentinel: ErrTokenAddressFormatV1,
		},
		{
			name:     "alphabet",
			address:  "not-a-Base58-address@@@",
			sentinel: ErrTokenAddressFormatV1,
		},
		{
			name:     "decoded length",
			address:  "1",
			sentinel: ErrTokenAddressFormatV1,
		},
		{
			name: "network",
			address: addressForPolicyVersion(
				0x6f,
				0x33,
			),
			sentinel: ErrTokenAddressNetworkV1,
		},
		{
			name:     "checksum",
			address:  corruptAddressChecksum(valid),
			sentinel: ErrTokenAddressChecksumV1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizeTokenAddressV1(
				test.address,
			)
			if err == nil {
				t.Fatal("invalid address accepted")
			}
			if !errors.Is(err, ErrInvalidTokenAddress) {
				t.Fatalf(
					"missing general address sentinel: %v",
					err,
				)
			}
			if !errors.Is(err, test.sentinel) {
				t.Fatalf(
					"missing dimension sentinel %v: %v",
					test.sentinel,
					err,
				)
			}
			if validTokenAddress(test.address) {
				t.Fatal("invalid address passed token boundary")
			}
		})
	}
}

func TestTokenAddressPolicyV1NormalizesButRequiresCanonicalStateKey(
	t *testing.T,
) {
	address := addressForPolicyVersion(
		TokenAddressNetworkVersionV1,
		0x44,
	)
	padded := " \t" + address + "\n "

	canonical, err := NormalizeTokenAddressV1(padded)
	if err != nil {
		t.Fatalf("normalization failed: %v", err)
	}
	if canonical != address {
		t.Fatalf(
			"normalization mismatch: got %q, want %q",
			canonical,
			address,
		)
	}
	if validTokenAddress(padded) {
		t.Fatal("non-canonical state key was accepted")
	}

	_, err = requireCanonicalTokenAddressV1(padded)
	if !errors.Is(err, ErrTokenAddressCanonicalV1) {
		t.Fatalf(
			"expected canonical rejection, got %v",
			err,
		)
	}
}

func TestTokenAddressPolicyV1ValidatedConstructors(
	t *testing.T,
) {
	owner := addressForPolicyVersion(
		TokenAddressNetworkVersionV1,
		0x55,
	)

	base, err := NewValidatedTokenV1(
		"VAL",
		"Validated Token",
		8,
		100,
		owner,
	)
	if err != nil || base.Owner() != owner {
		t.Fatalf(
			"validated Token constructor failed: %v",
			err,
		)
	}

	qc20, err := NewValidatedQC20TokenV1(
		"Validated Fixed Token",
		"VFT",
		100,
		8,
		owner,
	)
	if err != nil ||
		qc20.Owner() != owner ||
		qc20.BalanceOf(owner) != 100 {
		t.Fatalf(
			"validated QC20 constructor failed: %v",
			err,
		)
	}

	qc721, err := NewValidatedQC721TokenV1(
		"Validated Collection",
		"VNFT",
		owner,
	)
	if err != nil || qc721.Owner() != owner {
		t.Fatalf(
			"validated QC721 constructor failed: %v",
			err,
		)
	}

	for name, constructor := range map[string]func() error{
		"Token": func() error {
			_, err := NewValidatedTokenV1(
				"BAD",
				"Bad Token",
				8,
				100,
				"owner",
			)
			return err
		},
		"QC20": func() error {
			_, err := NewValidatedQC20TokenV1(
				"Bad Fixed Token",
				"BFT",
				100,
				8,
				"owner",
			)
			return err
		},
		"QC721": func() error {
			_, err := NewValidatedQC721TokenV1(
				"Bad Collection",
				"BNFT",
				"owner",
			)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := constructor(); !errors.Is(err, ErrInvalidTokenAddress) {
				t.Fatalf(
					"invalid owner accepted: %v",
					err,
				)
			}
		})
	}
}

func TestTokenAddressPolicyV1BlocksLegacyInvalidOwnerMutation(
	t *testing.T,
) {
	recipient := addressForPolicyVersion(
		TokenAddressNetworkVersionV1,
		0x66,
	)

	base := New(
		"LEG",
		"Legacy Token",
		8,
		100,
		"owner",
	)
	if err := base.Mint("owner", recipient, 1); !errors.Is(err, ErrInvalidTokenAddress) {
		t.Fatalf("legacy invalid Token owner mutated state: %v", err)
	}
	if base.TotalSupply() != 0 ||
		base.BalanceOf(recipient) != 0 {
		t.Fatal("rejected Token mint changed state")
	}

	qc721 := NewQC721TokenWithOwner(
		"Legacy Collection",
		"LNFT",
		"owner",
	)
	if err := qc721.MintByOwner(
		"owner",
		recipient,
		1,
		"ipfs://qma-0013",
	); !errors.Is(err, ErrInvalidTokenAddress) {
		t.Fatalf("legacy invalid QC721 owner mutated state: %v", err)
	}
	if _, err := qc721.OwnerOf(1); err == nil {
		t.Fatal("rejected QC721 mint created a token")
	}
}

func TestTokenAddressPolicyV1PersistenceRejectsAtomically(
	t *testing.T,
) {
	owner := addressForPolicyVersion(
		TokenAddressNetworkVersionV1,
		0x77,
	)

	t.Run("Token", func(t *testing.T) {
		target := New(
			"KEEP",
			"Keep Token",
			8,
			100,
			owner,
		)
		if err := target.Mint(owner, owner, 7); err != nil {
			t.Fatalf("prepare Token target: %v", err)
		}

		invalid := New(
			"BAD",
			"Bad Token",
			8,
			100,
			"owner",
		)
		raw, err := json.Marshal(invalid)
		if err != nil {
			t.Fatalf("marshal invalid Token: %v", err)
		}

		err = json.Unmarshal(raw, target)
		if !errors.Is(err, ErrInvalidTokenAddress) {
			t.Fatalf("invalid Token persistence accepted: %v", err)
		}
		if target.Owner() != owner ||
			target.TotalSupply() != 7 {
			t.Fatal("rejected Token persistence changed target")
		}
	})

	t.Run("QC20", func(t *testing.T) {
		target := NewQC20TokenWithOwner(
			"Keep Fixed Token",
			"KFT",
			7,
			8,
			owner,
		)
		invalid := NewQC20TokenWithOwner(
			"Bad Fixed Token",
			"BFT",
			9,
			8,
			"owner",
		)
		raw, err := json.Marshal(invalid)
		if err != nil {
			t.Fatalf("marshal invalid QC20: %v", err)
		}

		err = json.Unmarshal(raw, target)
		if !errors.Is(err, ErrInvalidTokenAddress) ||
			!errors.Is(err, ErrQC20PersistenceIntegrityV1) {
			t.Fatalf("invalid QC20 persistence accepted: %v", err)
		}
		if target.Owner() != owner ||
			target.TotalSupply() != 7 {
			t.Fatal("rejected QC20 persistence changed target")
		}
	})

	t.Run("QC721", func(t *testing.T) {
		target := NewQC721TokenWithOwner(
			"Keep Collection",
			"KNFT",
			owner,
		)
		invalid := NewQC721TokenWithOwner(
			"Bad Collection",
			"BNFT",
			"owner",
		)
		raw, err := json.Marshal(invalid)
		if err != nil {
			t.Fatalf("marshal invalid QC721: %v", err)
		}

		err = json.Unmarshal(raw, target)
		if !errors.Is(err, ErrInvalidTokenAddress) {
			t.Fatalf("invalid QC721 persistence accepted: %v", err)
		}
		if target.Owner() != owner {
			t.Fatal("rejected QC721 persistence changed target")
		}
	})
}

func TestTokenAddressPolicyV1AuthorizationRequiresCanonicalAddresses(
	t *testing.T,
) {
	caller := addressForPolicyVersion(
		TokenAddressNetworkVersionV1,
		0x88,
	)
	recipient := addressForPolicyVersion(
		TokenAddressNetworkVersionV1,
		0x99,
	)

	claim := TokenAuthorizationClaimV1{
		Version:   TokenAuthorizationVersionV1,
		Domain:    "qma-0013-domain",
		Operation: TokenOperationTransferV1,
		Caller:    caller,
		From:      caller,
		To:        recipient,
		Amount:    1,
		Nonce:     1,
	}

	if _, err := TokenAuthorizationDigestV1(claim); err != nil {
		t.Fatalf("canonical authorization rejected: %v", err)
	}

	claim.To = recipient + " "
	if _, err := TokenAuthorizationDigestV1(claim); !errors.Is(err, ErrTokenAuthorizationInvalid) {
		t.Fatalf(
			"non-canonical authorization address accepted: %v",
			err,
		)
	}
}
