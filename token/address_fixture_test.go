package token

import (
	"crypto/sha256"

	"quantumcoin/utils"
)

// canonicalAddressFixture converts historical human-readable
// test fixture labels into deterministic canonical QuantumCoin addresses.
//
// It is test-only. Production address validation remains strict.
func canonicalAddressFixture(value string) string {
	if value == "" {
		return ""
	}

	if canonical, err := NormalizeTokenAddressV1(value); err == nil &&
		canonical == value {
		return value
	}

	digest := sha256.Sum256(
		[]byte("qma0013-test-fixture:" + value),
	)

	payload := make(
		[]byte,
		tokenAddressPayloadSizeV1,
	)

	payload[0] = TokenAddressNetworkVersionV1

	copy(
		payload[1:],
		digest[:],
	)

	full := append(
		append([]byte(nil), payload...),
		utils.Checksum(payload)...,
	)

	return string(
		utils.Base58Encode(full),
	)
}
