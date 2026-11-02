package token

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const FungiblePersistenceVersionV1 uint8 = 1

var (
	ErrFungiblePersistenceEnvelopeV1 = errors.New(
		"token: invalid fungible persistence envelope",
	)
	ErrFungiblePersistenceVersionV1 = errors.New(
		"token: unsupported fungible persistence version",
	)
	ErrFungiblePersistenceModelV1 = errors.New(
		"token: unsupported fungible token model",
	)
	ErrFungiblePersistenceIntegrityV1 = errors.New(
		"token: fungible persistence integrity validation failed",
	)
	ErrQC20PersistenceIntegrityV1 = errors.New(
		"token: QC20 persistence integrity validation failed",
	)
)

type fungiblePersistenceEnvelopeV1 struct {
	Version uint8                `json:"version"`
	Model   FungibleTokenModelV1 `json:"model"`
	State   json.RawMessage      `json:"state"`
}

func fungiblePersistenceErrorV1(
	sentinel error,
	format string,
	arguments ...any,
) error {
	return fmt.Errorf(
		"%w: %s",
		sentinel,
		fmt.Sprintf(format, arguments...),
	)
}

func decodeJSONObjectFieldsV1(
	data []byte,
) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(
		bytes.NewReader(data),
	)

	opening, err := decoder.Token()
	if err != nil {
		return nil, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceEnvelopeV1,
			"decode JSON object: %v",
			err,
		)
	}

	if opening != json.Delim('{') {
		return nil, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceEnvelopeV1,
			"expected JSON object",
		)
	}

	fields := make(map[string]json.RawMessage)

	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, fungiblePersistenceErrorV1(
				ErrFungiblePersistenceEnvelopeV1,
				"decode object key: %v",
				err,
			)
		}

		key, ok := keyToken.(string)
		if !ok {
			return nil, fungiblePersistenceErrorV1(
				ErrFungiblePersistenceEnvelopeV1,
				"object key is not a string",
			)
		}

		if _, exists := fields[key]; exists {
			return nil, fungiblePersistenceErrorV1(
				ErrFungiblePersistenceEnvelopeV1,
				"duplicate object field %q",
				key,
			)
		}

		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, fungiblePersistenceErrorV1(
				ErrFungiblePersistenceEnvelopeV1,
				"decode object field %q: %v",
				key,
				err,
			)
		}

		fields[key] = raw
	}

	closing, err := decoder.Token()
	if err != nil {
		return nil, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceEnvelopeV1,
			"close JSON object: %v",
			err,
		)
	}

	if closing != json.Delim('}') {
		return nil, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceEnvelopeV1,
			"JSON object was not closed",
		)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fungiblePersistenceErrorV1(
				ErrFungiblePersistenceEnvelopeV1,
				"trailing JSON value",
			)
		}

		return nil, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceEnvelopeV1,
			"decode trailing JSON: %v",
			err,
		)
	}

	return fields, nil
}

func decodeFungiblePersistenceEnvelopeV1(
	data []byte,
) (fungiblePersistenceEnvelopeV1, error) {
	var envelope fungiblePersistenceEnvelopeV1

	fields, err := decodeJSONObjectFieldsV1(data)
	if err != nil {
		return envelope, err
	}

	for key := range fields {
		switch key {
		case "version", "model", "state":
		default:
			return envelope, fungiblePersistenceErrorV1(
				ErrFungiblePersistenceEnvelopeV1,
				"unknown envelope field %q",
				key,
			)
		}
	}

	for _, required := range []string{
		"version",
		"model",
		"state",
	} {
		if _, exists := fields[required]; !exists {
			return envelope, fungiblePersistenceErrorV1(
				ErrFungiblePersistenceEnvelopeV1,
				"missing envelope field %q",
				required,
			)
		}
	}

	if err := json.Unmarshal(
		fields["version"],
		&envelope.Version,
	); err != nil {
		return envelope, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceEnvelopeV1,
			"invalid version: %v",
			err,
		)
	}

	if err := json.Unmarshal(
		fields["model"],
		&envelope.Model,
	); err != nil {
		return envelope, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceModelV1,
			"invalid model: %v",
			err,
		)
	}

	envelope.State = append(
		json.RawMessage(nil),
		fields["state"]...,
	)

	if envelope.Version != FungiblePersistenceVersionV1 {
		return envelope, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceVersionV1,
			"version %d",
			envelope.Version,
		)
	}

	if !validFungibleTokenModelV1(envelope.Model) {
		return envelope, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceModelV1,
			"model %q",
			envelope.Model,
		)
	}

	if bytes.Equal(
		bytes.TrimSpace(envelope.State),
		[]byte("null"),
	) {
		return envelope, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceEnvelopeV1,
			"state is null",
		)
	}

	return envelope, nil
}

func validateFungibleStateSchemaV1(
	model FungibleTokenModelV1,
	state []byte,
) error {
	fields, err := decodeJSONObjectFieldsV1(state)
	if err != nil {
		return fungiblePersistenceErrorV1(
			ErrFungiblePersistenceIntegrityV1,
			"invalid %s state object: %v",
			model,
			err,
		)
	}

	var required []string
	var allowed map[string]struct{}

	switch model {
	case FungibleTokenModelControlledEmissionV1:
		required = []string{
			"symbol",
			"name",
			"decimals",
			"max_supply",
			"total_supply",
			"owner",
			"balances",
		}
		allowed = map[string]struct{}{
			"symbol":               {},
			"name":                 {},
			"decimals":             {},
			"max_supply":           {},
			"total_supply":         {},
			"owner":                {},
			"balances":             {},
			"authorization_nonces": {},
			"miner_token_blocks":   {},
		}

	case FungibleTokenModelFixedSupplyV1:
		required = []string{
			"Name",
			"Symbol",
			"Owner",
			"MaxSupply",
			"TotalSupply",
			"Decimals",
			"Balances",
			"Allowances",
		}
		allowed = map[string]struct{}{
			"Name":                {},
			"Symbol":              {},
			"Owner":               {},
			"MaxSupply":           {},
			"TotalSupply":         {},
			"Decimals":            {},
			"Balances":            {},
			"Allowances":          {},
			"AuthorizationNonces": {},
		}

	default:
		return fungiblePersistenceErrorV1(
			ErrFungiblePersistenceModelV1,
			"model %q",
			model,
		)
	}

	for key := range fields {
		if _, exists := allowed[key]; !exists {
			return fungiblePersistenceErrorV1(
				ErrFungiblePersistenceIntegrityV1,
				"field %q does not belong to %s",
				key,
				model,
			)
		}
	}

	for _, key := range required {
		if _, exists := fields[key]; !exists {
			return fungiblePersistenceErrorV1(
				ErrFungiblePersistenceIntegrityV1,
				"missing %s state field %q",
				model,
				key,
			)
		}
	}

	return nil
}

// MarshalFungibleTokenV1 writes a versioned model-tagged envelope while
// preserving the existing inner Token and QC20 JSON contracts unchanged.
func MarshalFungibleTokenV1(
	subject FungibleTokenReadV1,
) ([]byte, error) {
	var (
		model FungibleTokenModelV1
		state []byte
		err   error
	)

	switch typed := subject.(type) {
	case *Token:
		if typed == nil {
			return nil, fungiblePersistenceErrorV1(
				ErrFungiblePersistenceEnvelopeV1,
				"nil Token",
			)
		}

		model = FungibleTokenModelControlledEmissionV1
		state, err = json.Marshal(typed)

	case *QC20Token:
		if typed == nil {
			return nil, fungiblePersistenceErrorV1(
				ErrFungiblePersistenceEnvelopeV1,
				"nil QC20Token",
			)
		}

		model = FungibleTokenModelFixedSupplyV1
		state, err = json.Marshal(typed)

	default:
		return nil, fungiblePersistenceErrorV1(
			ErrFungiblePersistenceModelV1,
			"unsupported concrete token type",
		)
	}

	if err != nil {
		return nil, err
	}

	if err := validateFungibleStateSchemaV1(
		model,
		state,
	); err != nil {
		return nil, err
	}

	return json.Marshal(fungiblePersistenceEnvelopeV1{
		Version: FungiblePersistenceVersionV1,
		Model:   model,
		State:   state,
	})
}

// UnmarshalFungibleTokenV1 selects the concrete token type from the trusted
// envelope identity, validates its exact v1 schema and returns a read-only
// common view. It never mutates an existing token instance.
func UnmarshalFungibleTokenV1(
	data []byte,
) (
	result FungibleTokenReadV1,
	resultErr error,
) {
	model := FungibleTokenModelV1("")

	defer func() {
		outcome := "LOADED"
		if resultErr != nil {
			outcome = "REJECTED"
		}

		observeFungiblePersistenceV1(
			model,
			outcome,
			resultErr,
		)
	}()

	envelope, err :=
		decodeFungiblePersistenceEnvelopeV1(data)

	model = envelope.Model

	if err != nil {
		resultErr = err
		return nil, resultErr
	}

	if err := validateFungibleStateSchemaV1(
		envelope.Model,
		envelope.State,
	); err != nil {
		resultErr = err
		return nil, resultErr
	}

	switch envelope.Model {
	case FungibleTokenModelControlledEmissionV1:
		var subject Token

		if err := json.Unmarshal(
			envelope.State,
			&subject,
		); err != nil {
			resultErr = fungiblePersistenceErrorV1(
				ErrFungiblePersistenceIntegrityV1,
				"decode TOKEN state: %v",
				err,
			)
			return nil, resultErr
		}

		if err := validateLoadedRegistryTokenV1(
			subject.Symbol(),
			&subject,
			make(map[string]struct{}),
		); err != nil {
			resultErr = fungiblePersistenceErrorV1(
				ErrFungiblePersistenceIntegrityV1,
				"validate TOKEN state: %v",
				err,
			)
			return nil, resultErr
		}

		return &subject, nil

	case FungibleTokenModelFixedSupplyV1:
		var subject QC20Token

		if err := json.Unmarshal(
			envelope.State,
			&subject,
		); err != nil {
			resultErr = fungiblePersistenceErrorV1(
				ErrFungiblePersistenceIntegrityV1,
				"decode QC20 state: %v",
				err,
			)
			return nil, resultErr
		}

		return &subject, nil

	default:
		resultErr = fungiblePersistenceErrorV1(
			ErrFungiblePersistenceModelV1,
			"model %q",
			envelope.Model,
		)
		return nil, resultErr
	}
}

func qc20PersistenceIntegrityErrorV1(
	format string,
	arguments ...any,
) error {
	return fungiblePersistenceErrorV1(
		ErrQC20PersistenceIntegrityV1,
		format,
		arguments...,
	)
}

func qc20PersistenceAddressIntegrityErrorV1(
	field string,
) error {
	return fmt.Errorf(
		"%w: %w",
		qc20PersistenceIntegrityErrorV1(
			"invalid %s",
			field,
		),
		ErrInvalidTokenAddress,
	)
}

func normalizeAndValidateQC20PersistenceV1(
	persisted *qc20PersistenceV1,
) error {
	if persisted == nil {
		return qc20PersistenceIntegrityErrorV1(
			"nil persisted state",
		)
	}

	trimmedName := strings.TrimSpace(persisted.Name)
	if trimmedName == "" || trimmedName != persisted.Name {
		return qc20PersistenceIntegrityErrorV1(
			"invalid token name",
		)
	}

	trimmedSymbol := strings.TrimSpace(persisted.Symbol)
	if trimmedSymbol == "" ||
		trimmedSymbol != persisted.Symbol {
		return qc20PersistenceIntegrityErrorV1(
			"invalid token symbol",
		)
	}

	if persisted.Owner != "" {
		if !validTokenAddress(persisted.Owner) {
			return qc20PersistenceAddressIntegrityErrorV1(
				"token owner",
			)
		}
	} else if persisted.TotalSupply != 0 {
		return qc20PersistenceIntegrityErrorV1(
			"issued supply requires an owner",
		)
	}

	if persisted.TotalSupply > persisted.MaxSupply {
		return qc20PersistenceIntegrityErrorV1(
			"total supply %d exceeds max supply %d",
			persisted.TotalSupply,
			persisted.MaxSupply,
		)
	}

	if persisted.Balances == nil {
		persisted.Balances =
			make(map[string]uint64)
	}

	if persisted.Allowances == nil {
		persisted.Allowances =
			make(map[string]map[string]uint64)
	}

	if persisted.AuthorizationNonces == nil {
		persisted.AuthorizationNonces =
			make(map[string]uint64)
	}

	var balanceTotal uint64

	for address, balance := range persisted.Balances {
		if !validTokenAddress(address) {
			return qc20PersistenceAddressIntegrityErrorV1(
				"balance address",
			)
		}

		nextTotal, err := checkedAddUint64(
			balanceTotal,
			balance,
		)
		if err != nil {
			return qc20PersistenceIntegrityErrorV1(
				"balance total overflow",
			)
		}

		balanceTotal = nextTotal
	}

	if balanceTotal != persisted.TotalSupply {
		return qc20PersistenceIntegrityErrorV1(
			"balance total %d does not match total supply %d",
			balanceTotal,
			persisted.TotalSupply,
		)
	}

	for owner, allowances := range persisted.Allowances {
		if !validTokenAddress(owner) {
			return qc20PersistenceAddressIntegrityErrorV1(
				"allowance owner",
			)
		}

		if allowances == nil {
			allowances = make(map[string]uint64)
			persisted.Allowances[owner] = allowances
		}

		for spender := range allowances {
			if !validTokenAddress(spender) {
				return qc20PersistenceAddressIntegrityErrorV1(
					"allowance spender",
				)
			}
		}
	}

	for caller := range persisted.AuthorizationNonces {
		if !validTokenAddress(caller) {
			return qc20PersistenceAddressIntegrityErrorV1(
				"authorization nonce caller",
			)
		}
	}

	return nil
}
