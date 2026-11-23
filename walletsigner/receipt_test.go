package walletsigner

import (
	"bytes"
	"encoding/json"
	"net/http"
	"quantumcoin/blockchain"
	"strings"
	"testing"
)

func TestSignedReceiptRequiresExactPublicIdentity(t *testing.T) {
	id := strings.Repeat("a", 64)
	if got, err := ValidateSendResponseV1(http.StatusOK, []byte(`{"success":true,"txid":"`+id+`"}`), id); err != nil || got != id {
		t.Fatal(got, err)
	}
	for _, body := range []string{
		`{"success":true}`, `{"success":true,"txid":"abc123"}`,
		`{"success":false,"success":true,"txid":"` + id + `"}`,
		`{"success":false,"SUCCESS":true,"txid":"` + id + `"}`,
		`{"success":true,"txid":"` + id + `"} {}`,
		`{"success":true,"txid":"` + strings.Repeat("b", 64) + `"}`,
		`{"success":true,"txid":"` + id + `","priv_hex":"forbidden"}`,
		`null`,
	} {
		if _, err := ValidateSendResponseV1(http.StatusOK, []byte(body), id); err == nil {
			t.Fatal("unsafe receipt accepted")
		}
	}
	if _, err := ValidateSendResponseV1(http.StatusUnauthorized, []byte(`{"success":true,"txid":"`+id+`"}`), id); err == nil {
		t.Fatal("HTTP rejection accepted")
	}
}

func TestSignedEnvelopeRejectsAmbiguousOrSecretFields(t *testing.T) {
	tx := &blockchain.Transaction{ID: bytes.Repeat([]byte{1}, 32), Inputs: []blockchain.TransactionInput{{}}, Outputs: []blockchain.TransactionOutput{{Amount: 1}}}
	raw, err := json.Marshal(map[string]any{"transaction": tx})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = SignedTransactionIDV1(raw); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		`{"transaction":null}`, `{"transaction":{"Sender":"unverified-placeholder"}}`,
		string(raw) + ` {}`,
		strings.Replace(string(raw), `"transaction":`, `"transaction":null,"TRANSACTION":`, 1),
		strings.Replace(string(raw), `"ID":`, `"ID":null,"id":`, 1),
		strings.Replace(string(raw), `"Inputs":`, `"priv_hex":"forbidden","Inputs":`, 1),
	} {
		if _, err := SignedTransactionIDV1([]byte(body)); err == nil {
			t.Fatal("invalid signer envelope accepted")
		}
	}
	if _, err := SignedTransactionIDV1(bytes.Repeat([]byte{' '}, maxEnvelope+1)); err == nil {
		t.Fatal("oversized envelope accepted")
	}
}
