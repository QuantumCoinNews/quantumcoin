package p2p

import (
	"strings"
	"testing"
	"time"

	"quantumcoin/blockchain"
)

func TestNewServerV1RejectsNilBlockchain(
	t *testing.T,
) {
	_, err := NewServerV1(nil)

	if err == nil {
		t.Fatal(
			"nil blockchain was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"blockchain is unavailable",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestNormalizeListenAddressV1(
	t *testing.T,
) {
	addr, err :=
		normalizeListenAddressV1("8333")
	if err != nil {
		t.Fatalf(
			"normalize failed: %v",
			err,
		)
	}

	if addr != ":8333" {
		t.Fatalf(
			"normalized address = %q, want :8333",
			addr,
		)
	}
}

func TestServerV1StartStopLifecycle(
	t *testing.T,
) {
	bc := blockchain.NewBlockchain(
		1,
		0,
	)

	server, err := NewServerV1(bc)
	if err != nil {
		t.Fatalf(
			"new server: %v",
			err,
		)
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- server.Serve(
			"127.0.0.1:0",
		)
	}()

	deadline :=
		time.Now().Add(
			2 * time.Second,
		)

	for !server.IsRunning() {
		if time.Now().After(deadline) {
			t.Fatal(
				"server did not start",
			)
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	if server.ListenAddress() == "" {
		t.Fatal(
			"running server has empty listen address",
		)
	}

	if err := server.Close(); err != nil {
		t.Fatalf(
			"close server: %v",
			err,
		)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf(
				"Serve returned error after Close: %v",
				err,
			)
		}

	case <-time.After(
		2 * time.Second,
	):
		t.Fatal(
			"Serve did not exit after Close",
		)
	}

	if server.IsRunning() {
		t.Fatal(
			"server remained running after Close",
		)
	}
}
