// Package api contains QuantumCoin project code for this module.
package api

import (
	"encoding/json"
	"net/http"
)

// AddressProvider main.go tarafından set edilir:
//
//	api.AddressProvider = func() string { return w.Address }
var AddressProvider func() string

// RegisterWalletRoutes documents the QuantumCoin component used by this module.
// RegisterWalletRoutes, /api/wallet/address endpoint'ini mux'a ekler.
func RegisterWalletRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/wallet/address", func(w http.ResponseWriter, r *http.Request) {
		addr := ""
		if AddressProvider != nil {
			addr = AddressProvider()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"address": addr,
		})
	})
}
