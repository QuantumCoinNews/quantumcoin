package config

import (
	"encoding/json"
	"errors"
	"io"
)

// rejectDuplicateConfigKeysV1 prevents silent last-key-wins config changes.
// It resets the stream so the existing typed/unknown-field decoder remains
// authoritative. Diagnostics do not echo configuration values.
func rejectDuplicateConfigKeysV1(reader io.ReadSeeker) error {
	decoder := json.NewDecoder(reader)
	rootToken := true
	var visit func() error
	visit = func() error {
		token, err := decoder.Token()
		if err != nil {
			return errors.New("invalid configuration JSON")
		}
		if rootToken {
			rootToken = false
			if token != json.Delim('{') {
				return errors.New("configuration must be a JSON object")
			}
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				token, err := decoder.Token()
				if err != nil {
					return errors.New("invalid configuration JSON")
				}
				key, ok := token.(string)
				if !ok {
					return errors.New("invalid configuration key")
				}
				if seen[key] {
					return errors.New("duplicate configuration key")
				}
				seen[key] = true
				if err := visit(); err != nil {
					return err
				}
			}
		case '[':
			for decoder.More() {
				if err := visit(); err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid configuration JSON")
		}
		if _, err := decoder.Token(); err != nil {
			return errors.New("invalid configuration JSON")
		}
		return nil
	}
	if err := visit(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("multiple configuration JSON values")
	}
	_, err := reader.Seek(0, io.SeekStart)
	return err
}
