package token

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Token JSON must have a single interpretation before typed decoding can
// apply last-value-wins or case-insensitive field matching.
func validateUniqueJSONValueV1(decoder *json.Decoder, depth int) error {
	if depth > 64 {
		return errors.New("token JSON nesting limit exceeded")
	}
	opening, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := opening.(json.Delim)
	if !ok {
		return nil
	}
	if delim != '{' && delim != '[' {
		return errors.New("unexpected token JSON delimiter")
	}
	seen := make(map[string]bool)
	for decoder.More() {
		if delim == '{' {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return errors.New("duplicate or invalid token JSON key")
			}
			seen[name] = true
		}
		if err := validateUniqueJSONValueV1(decoder, depth+1); err != nil {
			return err
		}
	}
	closing, err := decoder.Token()
	if err != nil {
		return err
	}
	if (delim == '{' && closing != json.Delim('}')) || (delim == '[' && closing != json.Delim(']')) {
		return errors.New("unclosed token JSON value")
	}
	return nil
}

func validateUniqueJSONV1(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := validateUniqueJSONValueV1(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing token JSON content")
	}
	return nil
}

func registrySchemaFieldsV1(data []byte, required, optional []string) (map[string]json.RawMessage, error) {
	fields, err := decodeJSONObjectFieldsV1(data)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]bool)
	for _, name := range required {
		allowed[name] = true
		value, exists := fields[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("missing or null registry field %q", name)
		}
	}
	for _, name := range optional {
		allowed[name] = true
	}
	for name := range fields {
		if !allowed[name] {
			return nil, fmt.Errorf("unknown or noncanonical registry field %q", name)
		}
	}
	return fields, nil
}

func decodeRegistryStateV1(data []byte) (tokens map[string]*Token, resultErr error) {
	defer func() {
		if resultErr != nil {
			resultErr = fmt.Errorf("%w: decode registry: %w", ErrRegistryLoadIntegrityV1, resultErr)
		}
	}()
	if err := validateUniqueJSONV1(data); err != nil {
		return nil, err
	}
	envelope, err := decodeJSONObjectFieldsV1(data)
	if err != nil {
		return nil, err
	}
	rawTokens, exists := envelope["tokens"]
	if !exists || len(envelope) != 1 {
		return nil, errors.New("registry requires exactly the tokens field")
	}
	if !bytes.Equal(bytes.TrimSpace(rawTokens), []byte("null")) {
		entries, err := decodeJSONObjectFieldsV1(rawTokens)
		if err != nil {
			return nil, err
		}
		for _, raw := range entries {
			fields, err := registrySchemaFieldsV1(raw,
				[]string{"symbol", "name", "decimals", "max_supply", "total_supply", "owner"},
				[]string{"balances", "authorization_nonces", "miner_token_blocks", "authorization_identity"})
			if err != nil {
				return nil, err
			}
			// Legacy zero-supply records may omit maps; scalar supply and
			// identity fields must be explicit, including an unlimited cap 0.
			if records, ok := fields["miner_token_blocks"]; ok && !bytes.Equal(bytes.TrimSpace(records), []byte("null")) {
				blocks, err := decodeJSONObjectFieldsV1(records)
				if err != nil {
					return nil, err
				}
				for _, record := range blocks {
					if _, err := registrySchemaFieldsV1(record, []string{"version", "height", "block_hash", "miner", "reward", "status"}, nil); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	var decoded registryTokenMapV1
	if err := json.Unmarshal(rawTokens, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}
