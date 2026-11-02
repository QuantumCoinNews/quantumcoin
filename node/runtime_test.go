package node

import (
	"net/http"
	"testing"

	"quantumcoin/blockchain"
	"quantumcoin/config"
)

func TestNewRuntimeOwnsProvidedComponentsV1(
	t *testing.T,
) {
	bc := blockchain.NewBlockchain(1, 0)
	cfg := &config.Config{}

	runtime := NewRuntime(
		bc,
		cfg,
	)
	if runtime == nil {
		t.Fatal("runtime is nil")
	}

	gotBC, gotConfig :=
		runtime.Snapshot()

	if gotBC != bc {
		t.Fatal("runtime blockchain does not match provided blockchain")
	}

	if gotConfig != cfg {
		t.Fatal("runtime config does not match provided config")
	}
}

func TestRuntimeAPIReadOnlyModeV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil,
	)

	if runtime.IsAPIReadOnly() {
		t.Fatal("new runtime unexpectedly starts read-only")
	}

	runtime.SetAPIReadOnly(true)

	if !runtime.IsAPIReadOnly() {
		t.Fatal("runtime did not enter read-only mode")
	}

	runtime.SetAPIReadOnly(false)

	if runtime.IsAPIReadOnly() {
		t.Fatal("runtime did not leave read-only mode")
	}
}

func TestRuntimeStoresHTTPServerV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil,
	)

	server := &http.Server{
		Addr: ":8080",
	}

	runtime.SetHTTPServer(server)

	if got := runtime.HTTP(); got != server {
		t.Fatal("runtime HTTP server does not match provided server")
	}
}

func TestNilRuntimeAccessorsAreSafeV1(
	t *testing.T,
) {
	var runtime *Runtime

	bc, cfg := runtime.Snapshot()

	if bc != nil ||
		cfg != nil {
		t.Fatal("nil runtime returned non-nil snapshot values")
	}

	if runtime.HTTP() != nil {
		t.Fatal("nil runtime returned an HTTP server")
	}

	if runtime.IsAPIReadOnly() {
		t.Fatal("nil runtime unexpectedly reports read-only mode")
	}

	runtime.SetHTTPServer(
		&http.Server{},
	)
	runtime.SetAPIReadOnly(true)
}
