// TOKEN_SAFE_BOUNDARY_V1

// Package token contains QuantumCoin project code for this module.
package token

import (
	"errors"
	"os"
	"strings"
)

const TokenSafeBoundaryVersion = "TOKEN_SAFE_BOUNDARY_V1"

var (
	ErrInvalidTokenAddress      = errors.New("token: invalid address")
	ErrTokenAmountZero          = errors.New("token: amount is zero")
	ErrTokenSupplyOverflow      = errors.New("token: supply overflow")
	ErrTokenBalanceOverflow     = errors.New("token: balance overflow")
	ErrTokenMaxSupplyExceeded   = errors.New("token: max supply exceeded")
	ErrTokenOwnerNotConfigured  = errors.New("token: owner not configured")
	ErrTokenMinerBridgeDisabled = errors.New("token: miner token bridge disabled by safe boundary")
)

func TokenMinerBridgeEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("QC_ENABLE_TOKEN_MINER_BRIDGE")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func validTokenAddress(addr string) bool {
	_, err := requireCanonicalTokenAddressV1(addr)
	return err == nil
}

func checkedAddUint64(a, b uint64) (uint64, error) {
	if b > ^uint64(0)-a {
		return 0, ErrTokenSupplyOverflow
	}
	return a + b, nil
}

func validateMintCommon(to string, amount uint64, currentSupply uint64, maxSupply uint64) error {
	if !validTokenAddress(to) {
		return ErrInvalidTokenAddress
	}
	if amount == 0 {
		return ErrTokenAmountZero
	}
	nextSupply, err := checkedAddUint64(currentSupply, amount)
	if err != nil {
		return err
	}
	if maxSupply > 0 && nextSupply > maxSupply {
		return ErrTokenMaxSupplyExceeded
	}
	return nil
}
