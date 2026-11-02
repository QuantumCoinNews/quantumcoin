package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func requireCanonicalHTTPError(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	wantStatus int,
	wantMessage string,
) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf(
			"status=%d want=%d body=%q",
			rec.Code,
			wantStatus,
			rec.Body.String(),
		)
	}

	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(
		strings.ToLower(got),
		"application/json",
	) {
		t.Fatalf("content-type=%q", got)
	}

	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(
		rec.Body.Bytes(),
		&envelope,
	); err != nil {
		t.Fatalf(
			"invalid JSON error response: %v body=%q",
			err,
			rec.Body.String(),
		)
	}

	if envelope.Success {
		t.Fatalf(
			"error response unexpectedly reports success: %q",
			rec.Body.String(),
		)
	}

	if envelope.Message != wantMessage {
		t.Fatalf(
			"message=%q want=%q",
			envelope.Message,
			wantMessage,
		)
	}
}

func TestWriteJSONEncodeFailureUsesCanonicalError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(
		rec,
		http.StatusOK,
		map[string]any{
			"unsupported": make(chan int),
		},
	)

	requireCanonicalHTTPError(
		t,
		rec,
		http.StatusInternalServerError,
		"failed to encode response",
	)
}

func TestCanonicalMethodErrorsUseSharedEnvelope(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		url     string
	}{
		{
			name:    "transaction status",
			handler: handleTxStatus,
			url:     "/api/tx/status",
		},
		{
			name:    "economy policy",
			handler: handleEconomyPolicy,
			url:     "/api/economy/policy",
		},
		{
			name:    "security summary",
			handler: handleSecuritySummary,
			url:     "/api/security/summary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(
				http.MethodPost,
				tc.url,
				nil,
			)

			tc.handler(rec, req)

			requireCanonicalHTTPError(
				t,
				rec,
				http.StatusMethodNotAllowed,
				"method not allowed",
			)
		})
	}
}

func TestTxStatusMissingIDUsesCanonicalError(t *testing.T) {
	rec := httptest.NewRecorder()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/tx/status",
		nil,
	)

	handleTxStatus(rec, req)

	requireCanonicalHTTPError(
		t,
		rec,
		http.StatusBadRequest,
		"missing id",
	)
}

func TestStakeStatusUnavailableIsNotSuccessfulHTTP(t *testing.T) {
	rec := httptest.NewRecorder()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/stake/status",
		nil,
	)

	handleStakeStatus(rec, req)

	requireCanonicalHTTPError(
		t,
		rec,
		http.StatusNotImplemented,
		"stake status endpoint coming soon",
	)
}

func TestActiveResponseSourcesAvoidLegacyWriters(t *testing.T) {
	checks := []struct {
		path      string
		forbidden []string
	}{
		{
			path: "main.go",
			forbidden: []string{
				"_ = json.NewEncoder(w).Encode(v)",
				`map[string]string{"error": "method not allowed"}`,
				`"ok":    false`,
				"_, _ = w.Write(",
			},
		},
		{
			path: "http_security.go",
			forbidden: []string{
				"http.Error(",
			},
		},
		{
			path: "testnet_label_api_v1.go",
			forbidden: []string{
				"http.Error(",
				"_ = json.NewEncoder(w).Encode(info)",
			},
		},
	}

	for _, check := range checks {
		src, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf(
				"read %s: %v",
				check.path,
				err,
			)
		}

		text := string(src)

		for _, forbidden := range check.forbidden {
			if strings.Contains(
				text,
				forbidden,
			) {
				t.Fatalf(
					"%s still contains legacy response pattern %q",
					check.path,
					forbidden,
				)
			}
		}
	}
}
