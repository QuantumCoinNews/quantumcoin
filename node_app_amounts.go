package main

import (
	"encoding/json"
	"fmt"
	qconsensus "quantumcoin/consensus"
	"strings"
)

// FlexibleAmount documents the QuantumCoin component used by this module.
// Exact decimal QC amounts for local wallet construction.
type FlexibleAmount string

func (v FlexibleAmount) String() string {
	return string(v)
}

// UnmarshalJSON preserves the exact user-facing QC token.
// No float64 or integer conversion occurs at the API boundary.
func parseFlexibleAmountV1(
	raw string,
) (FlexibleAmount, error) {
	value := strings.TrimSpace(raw)

	atoms, err := qconsensus.ParseQCToAtomsV1(value)
	if err != nil {
		return "", fmt.Errorf(
			"invalid QC amount: %w",
			err,
		)
	}

	if atoms <= 0 {
		return "", fmt.Errorf(
			"QC amount must be positive",
		)
	}

	return FlexibleAmount(value), nil
}

func (v *FlexibleAmount) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("nil FlexibleAmount")
	}

	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return fmt.Errorf("empty amount")
	}

	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}

		parsed, err := parseFlexibleAmountV1(s)
		if err != nil {
			return err
		}

		*v = parsed
		return nil
	}

	if (raw[0] < '0' || raw[0] > '9') &&
		raw[0] != '-' {
		return fmt.Errorf(
			"amount must be a JSON number or string",
		)
	}

	parsed, err := parseFlexibleAmountV1(raw)
	if err != nil {
		return err
	}

	*v = parsed
	return nil
}
