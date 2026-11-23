package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddressEndpointRegressionRetiredProviderNeverSucceeds(t *testing.T) {
	old := AddressProvider
	defer func() { AddressProvider = old }()
	for _, provider := range []func() string{nil, func() string { return "" }, func() string { return "invalid-address" }} {
		AddressProvider = provider
		mux := http.NewServeMux()
		RegisterWalletRoutes(mux)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/wallet/address", nil))
		var failure struct {
			Success *bool  `json:"success"`
			Message string `json:"message"`
		}
		if response.Code != 410 || json.Unmarshal(response.Body.Bytes(), &failure) != nil || failure.Success == nil || *failure.Success || failure.Message == "" {
			t.Errorf("retired route must fail closed: %d %s", response.Code, response.Body.String())
		}
	}
}

func TestAddressEndpointRetiredRouteNeverCallsProvider(t *testing.T) {
	old := AddressProvider
	defer func() { AddressProvider = old }()
	AddressProvider = func() string { t.Fatal("retired provider invoked"); return "" }
	mux := http.NewServeMux()
	RegisterWalletRoutes(mux)
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(method, "/api/wallet/address", nil))
		want := 405
		if method == "GET" {
			want = 410
		}
		if response.Code != want {
			t.Fatalf("%s status=%d want=%d", method, response.Code, want)
		}
	}
}
