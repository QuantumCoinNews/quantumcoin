package main

import (
	"strings"
	"testing"
)

func TestCanonicalAPIResponseErrorUsesJSONMessage(t *testing.T) {
	err := canonicalAPIResponseError(
		400,
		[]byte(`{"success":false,"message":"method not allowed"}`),
	)

	if err == nil {
		t.Fatal("expected HTTP error")
	}

	if !strings.Contains(
		err.Error(),
		"method not allowed",
	) {
		t.Fatalf("error=%q", err)
	}
}

func TestCanonicalAPIResponseErrorAcceptsLegacyErrorField(t *testing.T) {
	err := canonicalAPIResponseError(
		400,
		[]byte(`{"error":"legacy error"}`),
	)

	if err == nil {
		t.Fatal("expected HTTP error")
	}

	if !strings.Contains(
		err.Error(),
		"legacy error",
	) {
		t.Fatalf("error=%q", err)
	}
}

func TestCanonicalAPIResponseErrorRejectsSemanticFailure(t *testing.T) {
	err := canonicalAPIResponseError(
		200,
		[]byte(`{"success":false,"message":"operation failed"}`),
	)

	if err == nil {
		t.Fatal("expected semantic API failure")
	}

	if !strings.Contains(
		err.Error(),
		"operation failed",
	) {
		t.Fatalf("error=%q", err)
	}
}

func TestCanonicalAPIResponseErrorDoesNotEchoRawBody(t *testing.T) {
	raw := []byte(
		`<html><body>internal proxy failure secret-detail</body></html>`,
	)

	err := canonicalAPIResponseError(
		500,
		raw,
	)

	if err == nil {
		t.Fatal("expected HTTP error")
	}

	if strings.Contains(
		err.Error(),
		"secret-detail",
	) {
		t.Fatalf(
			"raw response leaked into error: %q",
			err,
		)
	}

	if err.Error() != "api request failed: http 500" {
		t.Fatalf("error=%q", err)
	}
}

func TestCanonicalAPIResponseErrorAcceptsSuccessfulResponse(t *testing.T) {
	if err := canonicalAPIResponseError(
		204,
		nil,
	); err != nil {
		t.Fatalf(
			"unexpected successful-response error: %v",
			err,
		)
	}
}
