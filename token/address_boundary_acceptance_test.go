package token

import (
	"errors"
	"strings"
	"testing"
)

func invalidAddressBoundaryCases() map[string]string {
	valid := canonicalAddressFixture("address-boundary-owner")
	return map[string]string{
		"empty": "", "placeholder": "owner", "whitespace": " \t", "padded": " " + valid,
		"unicode padding": "\u00a0" + valid, "checksum": corruptAddressChecksum(valid),
		"wrong version": addressForPolicyVersion(0x6f, 0x59), "extra leading zero": "1" + valid,
		"invalid alphabet": "0OIl@@", "embedded control": valid[:4] + "\x00" + valid[4:],
		"oversized": strings.Repeat("z", 4096),
	}
}

func TestTokenAddressLegacyConstructorsCannotIssueToInvalidOwners(t *testing.T) {
	for name, address := range invalidAddressBoundaryCases() {
		t.Run(name, func(t *testing.T) {
			base := New("TEST", "Test", 8, 100, address)
			fixed := NewQC20TokenWithOwner("Test", "TEST", 100, 8, address)
			nft := NewQC721TokenWithOwner("Test", "TEST", address)
			if base.TotalSupply() != 0 || len(base.BalancesSnapshotV1()) != 0 {
				t.Fatal("invalid owner received controlled supply")
			}
			if fixed.TotalSupply() != 0 || len(fixed.BalancesSnapshotV1()) != 0 {
				t.Fatal("legacy QC20 constructor issued supply to invalid owner")
			}
			if fixed.MaxSupply() != 100 || fixed.Owner() != address {
				t.Fatal("legacy diagnostic metadata or cap was changed")
			}
			if len(nft.OwnersSnapshotV1()) != 0 {
				t.Fatal("invalid owner received NFT")
			}
		})
	}
}

func TestTokenAddressValidatedConstructorsRejectAllInvalidDimensions(t *testing.T) {
	for name, address := range invalidAddressBoundaryCases() {
		t.Run(name, func(t *testing.T) {
			base, err := NewValidatedTokenV1("TEST", "Test", 8, 100, address)
			if base != nil || !errors.Is(err, ErrInvalidTokenAddress) {
				t.Fatal("validated controlled constructor accepted invalid address")
			}
			fixed, err := NewValidatedQC20TokenV1("Test", "TEST", 100, 8, address)
			if fixed != nil || !errors.Is(err, ErrInvalidTokenAddress) {
				t.Fatal("validated fixed constructor accepted invalid address")
			}
			nft, err := NewValidatedQC721TokenV1("Test", "TEST", address)
			if nft != nil || !errors.Is(err, ErrInvalidTokenAddress) {
				t.Fatal("validated NFT constructor accepted invalid address")
			}
		})
	}
}

func TestTokenAddressAuthorizationChecksEveryAddressField(t *testing.T) {
	owner := canonicalAddressFixture("address-boundary-owner")
	base := TokenAuthorizationClaimV1{Version: TokenAuthorizationVersionV1, Domain: "address-boundary", Operation: TokenOperationTransferV1, Caller: owner, From: owner, To: owner, Spender: owner, Amount: 1, Nonce: 1}
	if _, err := TokenAuthorizationDigestV1(base); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"caller", "from", "to", "spender"} {
		t.Run(field, func(t *testing.T) {
			for name, address := range invalidAddressBoundaryCases() {
				// Optional empty fields are checked by their operation-specific
				// contract; this checks malformed non-empty signing addresses.
				if address == "" {
					continue
				}
				t.Run(name, func(t *testing.T) {
					claim := base
					switch field {
					case "caller":
						claim.Caller = address
					case "from":
						claim.From = address
					case "to":
						claim.To = address
					case "spender":
						claim.Spender = address
					}
					if _, err := TokenAuthorizationDigestV1(claim); !errors.Is(err, ErrTokenAuthorizationInvalid) {
						t.Fatalf("invalid signing address accepted: %v", err)
					}
				})
			}
		})
	}
}
