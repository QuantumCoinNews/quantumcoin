package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBurnRequestContainsOnlySignedTransactionV1(t *testing.T) {
	data, err := json.Marshal(BurnRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "priv_hex") || !strings.Contains(string(data), "transaction") {
		t.Fatal("unsafe burn contract")
	}
}
func TestBurnHandlerUsesCanonicalSignedSubmissionV1(t *testing.T) {
	data, err := os.ReadFile("node_app_transactions.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, required := range []string{"serveSignedTransactionV1(w, r, true)", "blockchain.IsCanonicalBurnOutputV1(out)", "bc.VerifyTransaction(tx)", "bc.AddTransaction(tx)"} {
		if !strings.Contains(source, required) {
			t.Fatal("missing canonical burn submission", required)
		}
	}
	for _, forbidden := range []string{"tx.Sign(", "ImportPrivateKeyHex", "PrivHex"} {
		if strings.Contains(source, forbidden) {
			t.Fatal("HTTP burn has signing keys", forbidden)
		}
	}
}
