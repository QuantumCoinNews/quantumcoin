package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestCanonicalSendRequestAcceptsGUIPayload(
	t *testing.T,
) {
	raw := []byte(`{
"from":"GUI_FROM",
"to":"GUI_TO",
"amount":"7",
"priv_hex":"deadbeef"
}`)

	var req SendRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf(
			"canonical GUI payload rejected: %v",
			err,
		)
	}

	if req.From != "GUI_FROM" {
		t.Fatalf("from = %q", req.From)
	}

	if req.To != "GUI_TO" {
		t.Fatalf("to = %q", req.To)
	}

	if req.Amount.String() != "7" {
		t.Fatalf(
			"amount = %q, want %q",
			req.Amount.String(),
			"7",
		)
	}

	if req.PrivHex != "deadbeef" {
		t.Fatalf(
			"priv_hex = %q",
			req.PrivHex,
		)
	}
}

func TestCanonicalServerSignsGUIRequest(
	t *testing.T,
) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}

	source := string(data)

	required := []string{
		`"/api/tx/send",`,
		`var req SendRequest`,
		`wallet.ImportPrivateKeyHex(privHex)`,
		`tx.Sign(priv)`,
		`tx.Verify()`,
		`nodeRuntime.BroadcastP2P(p2p.TxMessage(tx))`,
	}

	for _, item := range required {
		if !strings.Contains(source, item) {
			t.Fatalf(
				"canonical send contract missing %q",
				item,
			)
		}
	}

	for _, forbidden := range []string{
		"type txDTO struct",
		`"/api/tx/build"`,
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf(
				"retired/noncanonical tx contract present %q",
				forbidden,
			)
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
		"type canonicalSendResponse struct",
		`apiBase()+"/api/tx/send"`,
		"validateCanonicalSendResponse",
		"if !out.Success",
		"apiSendTransactionV1(",
		"Özel fee canonical /api/tx/send",
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
