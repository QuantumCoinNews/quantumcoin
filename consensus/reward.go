package consensus

import (
	"fmt"
	"math"
)

// BlockRewardBreakdownV1 separates newly minted subsidy from verified fees.
// Fees are transferred value and are never counted as newly minted supply.
type BlockRewardBreakdownV1 struct {
	Subsidy       int64
	Fees          int64
	CoinbaseTotal int64
}

// ValidateBlockRewardV1 validates the deterministic monetary composition of
// a block after coinbase layout, subsidy and transaction fees are calculated.
func ValidateBlockRewardV1(
	height uint64,
	coinbaseTotal int64,
	subsidy int64,
	fees int64,
	totalSupply int64,
) (BlockRewardBreakdownV1, error) {
	var result BlockRewardBreakdownV1

	if height == 0 {
		if totalSupply > 0 && coinbaseTotal > totalSupply {
			return result, fmt.Errorf(
				"%s: genesis allocation exceeds total supply: allocation=%d supply=%d",
				Version,
				coinbaseTotal,
				totalSupply,
			)
		}

		result.Subsidy = coinbaseTotal
		result.CoinbaseTotal = coinbaseTotal
		return result, nil
	}

	if subsidy <= 0 {
		return result, fmt.Errorf(
			"%s: emission is complete at block height %d",
			Version,
			height,
		)
	}

	if fees < 0 {
		return result, fmt.Errorf(
			"%s: negative verified fees: %d",
			Version,
			fees,
		)
	}

	if fees > math.MaxInt64-subsidy {
		return result, fmt.Errorf(
			"%s: coinbase subsidy plus fees overflow",
			Version,
		)
	}

	expectedCoinbase := subsidy + fees

	if coinbaseTotal != expectedCoinbase {
		return result, fmt.Errorf(
			"%s: block coinbase mismatch at height %d: got %d want subsidy=%d plus fees=%d total=%d",
			Version,
			height,
			coinbaseTotal,
			subsidy,
			fees,
			expectedCoinbase,
		)
	}

	result.Subsidy = subsidy
	result.Fees = fees
	result.CoinbaseTotal = coinbaseTotal

	return result, nil
}
