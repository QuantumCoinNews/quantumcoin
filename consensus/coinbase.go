package consensus

import "fmt"

const coinbaseMaxInt64V1 int64 = 1<<63 - 1

type CoinbaseTransactionViewV1 struct {
	Coinbase      bool
	OutputAmounts []int64
}

type CoinbaseLayoutResultV1 struct {
	Count int
	Total int64
}

func ValidateCoinbaseLayoutV1(
	transactions []*CoinbaseTransactionViewV1,
	genesis bool,
) (CoinbaseLayoutResultV1, error) {
	var result CoinbaseLayoutResultV1

	if len(transactions) == 0 {
		return result, fmt.Errorf(
			"%s: block has no transactions",
			Version,
		)
	}

	for transactionIndex, transaction := range transactions {
		if transaction == nil {
			return CoinbaseLayoutResultV1{}, fmt.Errorf(
				"%s: block transaction[%d] is nil",
				Version,
				transactionIndex,
			)
		}

		if !transaction.Coinbase {
			continue
		}

		result.Count++

		if !genesis && transactionIndex != 0 {
			return CoinbaseLayoutResultV1{}, fmt.Errorf(
				"%s: coinbase must be the first transaction",
				Version,
			)
		}

		for outputIndex, amount := range transaction.OutputAmounts {
			if amount <= 0 {
				return CoinbaseLayoutResultV1{}, fmt.Errorf(
					"%s: coinbase output[%d] invalid amount: %d",
					Version,
					outputIndex,
					amount,
				)
			}

			if result.Total > coinbaseMaxInt64V1-amount {
				return CoinbaseLayoutResultV1{}, fmt.Errorf(
					"%s: coinbase total overflow",
					Version,
				)
			}

			result.Total += amount
		}
	}

	if genesis {
		if result.Count < 1 {
			return CoinbaseLayoutResultV1{}, fmt.Errorf(
				"%s: genesis requires at least one allocation transaction",
				Version,
			)
		}

		return result, nil
	}

	if result.Count != 1 {
		return CoinbaseLayoutResultV1{}, fmt.Errorf(
			"%s: block must contain exactly one coinbase transaction, got %d",
			Version,
			result.Count,
		)
	}

	return result, nil
}
