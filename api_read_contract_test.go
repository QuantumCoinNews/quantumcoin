package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnknownAPIPathReturnsJSONNotFound(t *testing.T) {
	mux := http.NewServeMux()
	registerAPINotFoundBoundary(mux)

	for _, requestPath := range []string{
		"/api",
		"/api/does-not-exist",
		"/api/unknown/nested",
	} {
		t.Run(requestPath, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				requestPath,
				nil,
			)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf(
					"path=%q status=%d, want %d; body=%q",
					requestPath,
					rec.Code,
					http.StatusNotFound,
					rec.Body.String(),
				)
			}

			contentType := rec.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Fatalf(
					"path=%q Content-Type=%q, want JSON",
					requestPath,
					contentType,
				)
			}

			if strings.Contains(
				strings.ToLower(rec.Body.String()),
				"<html",
			) {
				t.Fatalf(
					"path=%q unexpectedly returned HTML: %q",
					requestPath,
					rec.Body.String(),
				)
			}

			var payload map[string]any
			if err := json.Unmarshal(
				rec.Body.Bytes(),
				&payload,
			); err != nil {
				t.Fatalf(
					"path=%q invalid JSON: %v body=%q",
					requestPath,
					err,
					rec.Body.String(),
				)
			}

			if success, ok := payload["success"].(bool); !ok || success {
				t.Fatalf(
					"path=%q success=%v, want false",
					requestPath,
					payload["success"],
				)
			}
		})
	}
}

func TestAPINotFoundBoundaryDoesNotShadowKnownRoute(t *testing.T) {
	mux := http.NewServeMux()

	registerAPINotFoundBoundary(mux)

	mux.HandleFunc(
		"/api/known",
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/known",
		nil,
	)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"status=%d, want %d",
			rec.Code,
			http.StatusNoContent,
		)
	}
}

func TestReadOnlyAPIHandlersRejectMutationMethods(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{
			name:    "wallet_address",
			path:    "/api/wallet/address",
			handler: handleWalletAddress,
		},
		{
			name:    "ai_bonus",
			path:    "/api/ai/bonus",
			handler: handleAIBonus,
		},
		{
			name:    "ai_analysis",
			path:    "/api/ai/analysis",
			handler: handleAIAnalysis,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodPost,
				tc.path,
				strings.NewReader(`{}`),
			)
			rec := httptest.NewRecorder()

			tc.handler(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf(
					"status=%d, want %d body=%q",
					rec.Code,
					http.StatusMethodNotAllowed,
					rec.Body.String(),
				)
			}

			if got := rec.Header().Get("Allow"); got != http.MethodGet {
				t.Fatalf(
					"Allow=%q, want %q",
					got,
					http.MethodGet,
				)
			}
		})
	}
}

func TestMinerStatusContractUsesGet(t *testing.T) {
	configureMinerControlSecurity(t)

	getReq := newMinerControlRequest(
		http.MethodGet,
		"/api/miner/status",
	)
	addMinerControlHeaderToken(getReq)

	getRec := httptest.NewRecorder()

	if !requireDevMinerEndpointAllowedV1(
		getRec,
		getReq,
		"/api/miner/status",
	) {
		t.Fatalf(
			"valid GET miner status rejected: status=%d body=%q",
			getRec.Code,
			getRec.Body.String(),
		)
	}

	postReq := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/status",
	)
	addMinerControlHeaderToken(postReq)

	postRec := httptest.NewRecorder()

	if requireDevMinerEndpointAllowedV1(
		postRec,
		postReq,
		"/api/miner/status",
	) {
		t.Fatal("POST miner status request was accepted")
	}

	if postRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"POST status=%d, want %d body=%q",
			postRec.Code,
			http.StatusMethodNotAllowed,
			postRec.Body.String(),
		)
	}

	if got := postRec.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf(
			"POST Allow=%q, want %q",
			got,
			http.MethodGet,
		)
	}
}

func TestMinerStatusHandlerRejectsPostBeforeRuntimeAccess(t *testing.T) {
	configureMinerControlSecurity(t)

	req := newMinerControlRequest(
		http.MethodPost,
		"/api/miner/status",
	)
	addMinerControlHeaderToken(req)

	rec := httptest.NewRecorder()

	handleMinerStatus(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"status=%d, want %d body=%q",
			rec.Code,
			http.StatusMethodNotAllowed,
			rec.Body.String(),
		)
	}

	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf(
			"Allow=%q, want %q",
			got,
			http.MethodGet,
		)
	}
}
