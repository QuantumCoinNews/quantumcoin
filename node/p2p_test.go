package node

import (
	"testing"
	"time"

	"quantumcoin/blockchain"
)

func TestRuntimeP2PStartStopLifecycleV1(
	t *testing.T,
) {
	bc := blockchain.NewBlockchain(
		1,
		0,
	)

	runtime := NewRuntime(
		bc,
		nil)

	errCh := make(chan error, 1)

	go func() {
		errCh <- runtime.ServeP2P(
			"127.0.0.1:0",
		)
	}()

	deadline :=
		time.Now().Add(
			2 * time.Second,
		)

	for !runtime.IsP2PRunning() {
		if time.Now().After(deadline) {
			t.Fatal(
				"runtime p2p server did not start",
			)
		}

		time.Sleep(
			10 * time.Millisecond,
		)
	}

	if runtime.P2PListenAddress() == "" {
		t.Fatal(
			"running p2p server has empty listen address",
		)
	}

	if err := runtime.StopP2P(); err != nil {
		t.Fatalf(
			"stop p2p: %v",
			err,
		)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf(
				"ServeP2P returned error after stop: %v",
				err,
			)
		}

	case <-time.After(
		2 * time.Second,
	):
		t.Fatal(
			"ServeP2P did not exit after stop",
		)
	}

	if runtime.IsP2PRunning() {
		t.Fatal(
			"runtime p2p remained running",
		)
	}
}

func TestNilRuntimeP2PStateV1(
	t *testing.T,
) {
	var runtime *Runtime

	if runtime.IsP2PRunning() {
		t.Fatal(
			"nil runtime reported p2p running",
		)
	}

	if runtime.PeerCount() != 0 {
		t.Fatal(
			"nil runtime reported peers",
		)
	}

	if err := runtime.StopP2P(); err != nil {
		t.Fatalf(
			"nil runtime stop returned error: %v",
			err,
		)
	}
}
