// TOKEN_ADDRESS_POLICY_V1

package token

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"quantumcoin/utils"
)

const TokenAddressPolicyVersionV1 = "QC_TOKEN_ADDRESS_POLICY_V1"

const (
	TokenAddressNetworkVersionV1 byte = 0x00
	tokenAddressPayloadSizeV1         = 21
	tokenAddressChecksumSizeV1        = 4
	tokenAddressDecodedSizeV1         = tokenAddressPayloadSizeV1 +
		tokenAddressChecksumSizeV1
)

var (
	ErrTokenAddressFormatV1 = errors.New(
		"token: invalid address format",
	)
	ErrTokenAddressNetworkV1 = errors.New(
		"token: invalid address network",
	)
	ErrTokenAddressChecksumV1 = errors.New(
		"token: invalid address checksum",
	)
	ErrTokenAddressCanonicalV1 = errors.New(
		"token: non-canonical address",
	)
)

func tokenAddressPolicyErrorV1(
	cause error,
	message string,
) error {
	return fmt.Errorf(
		"%w: %w: %s",
		ErrInvalidTokenAddress,
		cause,
		message,
	)
}

// NormalizeTokenAddressV1 returns the unique QuantumCoin Base58Check
// representation for the current token-address network. Callers that mutate
// token state must additionally require the input itself to equal this value.
func NormalizeTokenAddressV1(
	address string,
) (string, error) {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return "", tokenAddressPolicyErrorV1(
			ErrTokenAddressFormatV1,
			"address is empty",
		)
	}

	decoded, err := utils.Base58Decode(
		[]byte(trimmed),
	)
	if err != nil {
		return "", tokenAddressPolicyErrorV1(
			ErrTokenAddressFormatV1,
			"Base58 decoding failed",
		)
	}

	if len(decoded) != tokenAddressDecodedSizeV1 {
		return "", tokenAddressPolicyErrorV1(
			ErrTokenAddressFormatV1,
			fmt.Sprintf(
				"decoded length %d, want %d",
				len(decoded),
				tokenAddressDecodedSizeV1,
			),
		)
	}

	if decoded[0] != TokenAddressNetworkVersionV1 {
		return "", tokenAddressPolicyErrorV1(
			ErrTokenAddressNetworkV1,
			fmt.Sprintf(
				"network version 0x%02x, want 0x%02x",
				decoded[0],
				TokenAddressNetworkVersionV1,
			),
		)
	}

	payload := decoded[:tokenAddressPayloadSizeV1]
	checksum := decoded[tokenAddressPayloadSizeV1:]
	expectedChecksum := utils.Checksum(payload)

	if !bytes.Equal(checksum, expectedChecksum) {
		return "", tokenAddressPolicyErrorV1(
			ErrTokenAddressChecksumV1,
			"checksum mismatch",
		)
	}

	canonical := string(utils.Base58Encode(decoded))
	if canonical == "" || canonical != trimmed {
		return "", tokenAddressPolicyErrorV1(
			ErrTokenAddressCanonicalV1,
			"Base58 round trip changed the address",
		)
	}

	return canonical, nil
}

func requireCanonicalTokenAddressV1(
	address string,
) (string, error) {
	canonical, err := NormalizeTokenAddressV1(address)
	if err != nil {
		return "", err
	}

	if canonical != address {
		return "", tokenAddressPolicyErrorV1(
			ErrTokenAddressCanonicalV1,
			"input is not the canonical representation",
		)
	}

	return canonical, nil
}
