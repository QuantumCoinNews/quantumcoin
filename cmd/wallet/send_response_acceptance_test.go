package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestSendResponseRejectsMissingMalformedAndAmbiguousReceipt(t *testing.T) {
	for _, body := range []string{
		`{"success":true}`,
		`{"success":true,"txid":"abc123"}`,
		`{"success":false,"success":true,"txid":"` + strings.Repeat("a", 64) + `"}`,
		`{"success":true,"txid":"` + strings.Repeat("a", 64) + `"} {}`,
	} {
		t.Run(body, func(t *testing.T) {
			if _, err := validateCanonicalSendResponse(http.StatusOK, []byte(body), ""); err == nil {
				t.Fatal("invalid receipt was shown as successful")
			}
		})
	}
}
