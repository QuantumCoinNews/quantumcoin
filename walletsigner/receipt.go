package walletsigner

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"quantumcoin/blockchain"
)

// decodePublicJSONV1 rejects duplicate (including case-alias) keys before
// decoding the public wire schema. It is not a consensus validation function.
func decodePublicJSONV1(raw []byte, out any) error {
	if len(raw) == 0 || len(raw) > maxEnvelope {
		return errors.New("invalid public JSON size")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	var scan func(int) error
	scan = func(depth int) error {
		if depth > 32 {
			return errors.New("public JSON nesting limit")
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{':
				seen := map[string]bool{}
				for d.More() {
					key, err := d.Token()
					if err != nil {
						return err
					}
					name, ok := key.(string)
					if !ok {
						return errors.New("invalid JSON key")
					}
					name = strings.ToLower(name)
					if seen[name] {
						return errors.New("ambiguous JSON key")
					}
					seen[name] = true
					if err = scan(depth + 1); err != nil {
						return err
					}
				}
			case '[':
				for d.More() {
					if err := scan(depth + 1); err != nil {
						return err
					}
				}
			default:
				return errors.New("invalid JSON delimiter")
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := scan(0); err != nil {
		return errors.New("invalid or ambiguous public JSON")
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing public JSON")
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return errors.New("invalid public wire schema")
	}
	return nil
}

// SignedTransactionIDV1 extracts the public ID without duplicating the node's
// signature, UTXO, network or consensus authority.
func SignedTransactionIDV1(raw json.RawMessage) (string, error) {
	var envelope struct {
		Transaction *blockchain.Transaction `json:"transaction"`
	}
	if err := decodePublicJSONV1(raw, &envelope); err != nil {
		return "", err
	}
	tx := envelope.Transaction
	if tx == nil || len(tx.ID) != 32 || tx.IsCoinbase() || len(tx.Outputs) == 0 {
		return "", errors.New("signed transaction envelope required")
	}
	return hex.EncodeToString(tx.ID), nil
}

// ValidateSendResponseV1 binds a node acknowledgement to the submitted ID.
// Acceptance means mempool admission, not confirmation or finality.
func ValidateSendResponseV1(status int, raw []byte, expectedID string) (string, error) {
	if status != http.StatusOK {
		return "", errors.New("node did not accept the signed transaction")
	}
	var out struct {
		Success bool   `json:"success"`
		TxID    string `json:"txid"`
		Message string `json:"message,omitempty"`
	}
	if err := decodePublicJSONV1(raw, &out); err != nil {
		return "", err
	}
	if !out.Success {
		return "", errors.New("node rejected the signed transaction")
	}
	id, err := hex.DecodeString(out.TxID)
	if err != nil || len(out.TxID) != 64 || len(id) != 32 {
		return "", errors.New("node returned an invalid transaction ID")
	}
	if expectedID != "" && !strings.EqualFold(out.TxID, expectedID) {
		return "", errors.New("node acknowledged a different transaction")
	}
	return hex.EncodeToString(id), nil
}
