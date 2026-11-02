package consensus

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	CanonicalQCDecimalsV1       = 6
	CanonicalAtomsPerQCV1 int64 = 1_000_000
)

var (
	ErrInvalidQCAmountV1   = errors.New("invalid QC amount")
	ErrQCAmountPrecisionV1 = errors.New("QC amount exceeds 6 decimal places")
	ErrQCAmountOverflowV1  = errors.New("QC amount overflows atom range")
)

func ParseQCToAtomsV1(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, ErrInvalidQCAmountV1
	}

	if strings.HasPrefix(value, "+") {
		value = value[1:]
	}

	if value == "" || strings.HasPrefix(value, "-") {
		return 0, ErrInvalidQCAmountV1
	}

	if strings.Count(value, ".") > 1 {
		return 0, ErrInvalidQCAmountV1
	}

	parts := strings.SplitN(value, ".", 2)
	wholePart := parts[0]
	if wholePart == "" {
		wholePart = "0"
	}

	for _, r := range wholePart {
		if r < '0' || r > '9' {
			return 0, ErrInvalidQCAmountV1
		}
	}

	fractionPart := ""
	if len(parts) == 2 {
		fractionPart = parts[1]
	}

	for _, r := range fractionPart {
		if r < '0' || r > '9' {
			return 0, ErrInvalidQCAmountV1
		}
	}

	if len(fractionPart) > CanonicalQCDecimalsV1 {
		return 0, ErrQCAmountPrecisionV1
	}

	whole, err := strconv.ParseInt(wholePart, 10, 64)
	if err != nil {
		return 0, ErrQCAmountOverflowV1
	}

	if whole > (int64(^uint64(0)>>1) / CanonicalAtomsPerQCV1) {
		return 0, ErrQCAmountOverflowV1
	}

	atoms := whole * CanonicalAtomsPerQCV1

	if fractionPart != "" {
		padded := fractionPart + strings.Repeat(
			"0",
			CanonicalQCDecimalsV1-len(fractionPart),
		)

		fraction, err := strconv.ParseInt(padded, 10, 64)
		if err != nil {
			return 0, ErrInvalidQCAmountV1
		}

		if atoms > int64(^uint64(0)>>1)-fraction {
			return 0, ErrQCAmountOverflowV1
		}

		atoms += fraction
	}

	return atoms, nil
}

func FormatAtomsAsQCV1(atoms int64) string {
	if atoms < 0 {
		return "-" + FormatAtomsAsQCV1(-atoms)
	}

	whole := atoms / CanonicalAtomsPerQCV1
	fraction := atoms % CanonicalAtomsPerQCV1

	if fraction == 0 {
		return strconv.FormatInt(whole, 10)
	}

	formattedFraction := fmt.Sprintf(
		"%0*d",
		CanonicalQCDecimalsV1,
		fraction,
	)

	formattedFraction = strings.TrimRight(
		formattedFraction,
		"0",
	)

	return fmt.Sprintf(
		"%d.%s",
		whole,
		formattedFraction,
	)
}

func WholeQCToAtomsV1(qc int64) (int64, error) {
	if qc < 0 {
		return 0, ErrInvalidQCAmountV1
	}

	maxInt64 := int64(^uint64(0) >> 1)
	if qc > maxInt64/CanonicalAtomsPerQCV1 {
		return 0, ErrQCAmountOverflowV1
	}

	return qc * CanonicalAtomsPerQCV1, nil
}
