package config

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
)

const FounderAddressPolicyVersionV1 = "QC_FOUNDER_ADDRESS_POLICY_V1"

const founderBase58AlphabetV1 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// ValidateFounderAddressBindingV1 is a mainnet activation gate. Local release
// and testnet profiles may continue without final founder addresses, but an
// explicitly activated mainnet process must use two valid and distinct QC
// addresses for the liquid and vesting allocations.
func ValidateFounderAddressBindingV1(c *Config) error {
	if c == nil {
		return fmt.Errorf(
			"%s: config cannot be nil",
			FounderAddressPolicyVersionV1,
		)
	}

	profile, err := ResolveNetworkProfileV1(c.ChainFile)
	if err != nil {
		return fmt.Errorf(
			"%s: resolve network profile: %w",
			FounderAddressPolicyVersionV1,
			err,
		)
	}

	if !profile.IsExplicitMainnet {
		return nil
	}

	liquid := strings.TrimSpace(c.PremineLiquidAddress)
	locked := strings.TrimSpace(c.PremineLockedAddress)

	if liquid == "" {
		return fmt.Errorf(
			"%s: premine_liquid_address is required for mainnet",
			FounderAddressPolicyVersionV1,
		)
	}

	if locked == "" {
		return fmt.Errorf(
			"%s: premine_locked_address is required for mainnet",
			FounderAddressPolicyVersionV1,
		)
	}

	if liquid == locked {
		return fmt.Errorf(
			"%s: liquid and locked founder addresses must be distinct",
			FounderAddressPolicyVersionV1,
		)
	}

	if !validateQCFounderAddressV1(liquid) {
		return fmt.Errorf(
			"%s: invalid premine_liquid_address",
			FounderAddressPolicyVersionV1,
		)
	}

	if !validateQCFounderAddressV1(locked) {
		return fmt.Errorf(
			"%s: invalid premine_locked_address",
			FounderAddressPolicyVersionV1,
		)
	}

	return nil
}

func validateQCFounderAddressV1(address string) bool {
	decoded, ok := decodeFounderBase58V1(strings.TrimSpace(address))
	if !ok || len(decoded) != 25 {
		return false
	}

	if decoded[0] != 0x00 {
		return false
	}

	payload := decoded[:21]
	actualChecksum := decoded[21:]

	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])

	return bytes.Equal(actualChecksum, second[:4])
}

func decodeFounderBase58V1(value string) ([]byte, bool) {
	if value == "" {
		return nil, false
	}

	result := big.NewInt(0)
	radix := big.NewInt(58)

	for i := 0; i < len(value); i++ {
		index := strings.IndexByte(
			founderBase58AlphabetV1,
			value[i],
		)
		if index < 0 {
			return nil, false
		}

		result.Mul(result, radix)
		result.Add(result, big.NewInt(int64(index)))
	}

	decoded := result.Bytes()
	leadingZeroes := 0

	for leadingZeroes < len(value) && value[leadingZeroes] == '1' {
		leadingZeroes++
	}

	out := make([]byte, leadingZeroes+len(decoded))
	copy(out[leadingZeroes:], decoded)

	return out, true
}
