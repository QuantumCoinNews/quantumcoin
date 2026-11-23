package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCanonicalSendRequestOnlyCarriesSignedPublicData(t *testing.T) {
	data, err := json.Marshal(SendRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "priv_hex") || strings.Contains(string(data), "amount") {
		t.Fatal("HTTP envelope contains signing intent or secrets")
	}
}

func TestCanonicalServerVerifiesWithoutSigningGUIRequest(t *testing.T) {
	data, err := os.ReadFile("node_app_transactions.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, required := range []string{"decoder.DisallowUnknownFields()", "bc.VerifyTransaction(tx)", "bc.AddTransaction(tx)", "nodeRuntime.BroadcastP2P(p2p.TxMessage(tx))"} {
		if !strings.Contains(source, required) {
			t.Fatal("signed submission boundary missing", required)
		}
	}
	for _, forbidden := range []string{".Sign(", "ImportPrivateKeyHex", "req.PrivHex"} {
		if strings.Contains(source, forbidden) {
			t.Fatal("HTTP server still signs or receives keys", forbidden)
		}
	}
}

func TestWalletUsesSingleCanonicalSendPath(
	t *testing.T,
) {
	data, err := os.ReadFile(
		"cmd/wallet/main_fyne.go",
	)
	if err != nil {
		t.Fatal(err)
	}

	source := string(data)

	required := []string{
		"type canonicalSendRequest struct",
		`apiBase()+"/api/tx/send"`,
		"validateCanonicalSendResponse",
		"walletsigner.ValidateSendResponseV1(",
		"apiSendTransactionV1(",
		"özel fee canonical /api/tx/send",
	}

	for _, item := range required {
		if !strings.Contains(source, item) {
			t.Fatalf(
				"wallet canonical send contract missing %q",
				item,
			)
		}
	}

	for _, forbidden := range []string{
		`apiPost("/api/send"`,
		`apiPost("/api/tx/send"`,
		`"fee":      strings.TrimSpace(feeEntry.Text)`,
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf(
				"wallet legacy send behavior remains %q",
				forbidden,
			)
		}
	}
}
