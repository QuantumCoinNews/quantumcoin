package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	qwallet "quantumcoin/wallet"
)

func TestBalanceEndpointRejectsMutationMethod(t *testing.T) {
	addr := qwallet.NewWallet().GetAddress()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/wallet/balance/"+addr,
		nil,
	)
	rec := httptest.NewRecorder()

	handleBalance(rec, req)

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

func TestBalanceEndpointRejectsInvalidAddressBeforeChainAccess(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/wallet/balance/not-a-valid-qc-address@@@",
		nil,
	)
	rec := httptest.NewRecorder()

	handleBalanceForChain(rec, req, nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"status=%d, want %d body=%q",
			rec.Code,
			http.StatusBadRequest,
			rec.Body.String(),
		)
	}
}

func TestBalanceEndpointReportsUnavailableBlockchainAs503(t *testing.T) {
	addr := qwallet.NewWallet().GetAddress()

	if !qwallet.ValidateAddress(addr) {
		t.Fatalf(
			"generated test address is invalid: %q",
			addr,
		)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/wallet/balance/"+addr,
		nil,
	)
	rec := httptest.NewRecorder()

	handleBalanceForChain(rec, req, nil)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"status=%d, want %d body=%q",
			rec.Code,
			http.StatusServiceUnavailable,
			rec.Body.String(),
		)
	}
}

func TestWalletAddressEndpointKeepsGetOnlyContract(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/wallet/address",
		nil,
	)
	rec := httptest.NewRecorder()

	handleWalletAddress(rec, req)

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

func TestGeneratedWalletAddressPassesCanonicalValidator(t *testing.T) {
	addr := qwallet.NewWallet().GetAddress()

	if addr == "" {
		t.Fatal("generated wallet address is empty")
	}

	if !qwallet.ValidateAddress(addr) {
		t.Fatalf(
			"generated wallet address failed validator: %q",
			addr,
		)
	}
}
