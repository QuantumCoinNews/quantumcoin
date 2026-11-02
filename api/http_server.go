// Package api contains compatibility helpers retained after standalone HTTP retirement.
//
// QuantumCoin's canonical HTTP API is assembled and served by package main.
// The former standalone package-api HTTP server is permanently retired.
package api

import (
	"errors"

	"quantumcoin/blockchain"
	"quantumcoin/config"
)

// ErrStandaloneHTTPRetiredV1 is returned whenever legacy code attempts to
// launch the retired package-api HTTP server.
var ErrStandaloneHTTPRetiredV1 = errors.New(
	"api: standalone HTTP server retired; use the canonical QuantumCoin node API",
)

// Init is retained for source compatibility with legacy callers.
//
// It no longer stores HTTP-server state because package api no longer owns
// a standalone server.
//
// Deprecated: standalone HTTP retirement retired the standalone package-api HTTP architecture.
func Init(
	_ *blockchain.Blockchain,
	_ any,
	_ *config.Config,
) {
}

// StartHTTP is retained for source compatibility and always fails closed.
// It never creates, binds, or starts a network listener.
//
// Deprecated: standalone HTTP retirement retired the standalone package-api HTTP architecture.
func StartHTTP(_ string) error {
	return ErrStandaloneHTTPRetiredV1
}
