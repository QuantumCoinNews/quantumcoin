package main

import (
	"strings"
	"testing"
)

func TestSendResponseRejectsHTTPFailure(
	t *testing.T,
) {
	_, err := validateCanonicalSendResponse(
		500,
		[]byte(`{"success":false,"message":"server error"}`),
		"",
	)

	if err == nil {
		t.Fatal("HTTP failure was accepted")
	}
}

func TestSendResponseRejectsSemanticFailure(
	t *testing.T,
) {
	_, err := validateCanonicalSendResponse(
		200,
		[]byte(`{
"success":false,
"txid":"",
"message":"verify failed"
}`),
		"",
	)

	if err == nil {
		t.Fatal("HTTP 200 success:false was accepted")
	}
}

func TestSendResponseRejectsInvalidJSON(
	t *testing.T,
) {
	_, err := validateCanonicalSendResponse(
		200,
		[]byte(`not-json`),
		"",
	)

	if err == nil {
		t.Fatal("invalid response JSON was accepted")
	}
}

func TestSendResponseAcceptsCanonicalSuccess(
	t *testing.T,
) {
	_, err := validateCanonicalSendResponse(
		200,
		[]byte(`{
"success":true,
"txid":"`+strings.Repeat("a", 64)+`",
"message":"accepted"
}`),
		strings.Repeat("a", 64),
	)

	if err != nil {
		t.Fatalf(
			"canonical success rejected: %v",
			err,
		)
	}
}
