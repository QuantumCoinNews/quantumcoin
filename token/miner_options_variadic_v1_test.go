package token

import (
	"errors"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/miner"
)

func TestMinerTokenBridgeOptionsV1MergesAllUserOptions(
	t *testing.T,
) {
	bridge := &MinerTokenBridge{}

	tickCount := 0
	errorCount := 0
	broadcastCount := 0

	sentinel := errors.New(
		"qma0014 sentinel",
	)

	opts := bridge.Options(
		miner.Options{
			OnTick: func() {
				tickCount++
			},
		},
		miner.Options{
			OnError: func(err error) {
				if !errors.Is(err, sentinel) {
					t.Fatalf(
						"OnError err = %v, want sentinel",
						err,
					)
				}

				errorCount++
			},
		},
		miner.Options{
			Interval: 37 * time.Millisecond,
			Broadcaster: func(
				*blockchain.Block,
			) {
				broadcastCount++
			},
			Animate: true,
		},
	)

	if opts.OnTick == nil {
		t.Fatal(
			"first variadic option OnTick was lost",
		)
	}

	if opts.OnError == nil {
		t.Fatal(
			"second variadic option OnError was lost",
		)
	}

	if opts.Broadcaster == nil {
		t.Fatal(
			"third variadic option Broadcaster was lost",
		)
	}

	if opts.Interval != 37*time.Millisecond {
		t.Fatalf(
			"Interval = %v, want %v",
			opts.Interval,
			37*time.Millisecond,
		)
	}

	if !opts.Animate {
		t.Fatal(
			"last variadic option Animate was lost",
		)
	}

	opts.OnTick()
	opts.OnError(sentinel)
	opts.Broadcaster(nil)

	if tickCount != 1 {
		t.Fatalf(
			"OnTick count = %d, want 1",
			tickCount,
		)
	}

	if errorCount != 1 {
		t.Fatalf(
			"OnError count = %d, want 1",
			errorCount,
		)
	}

	if broadcastCount != 1 {
		t.Fatalf(
			"Broadcaster count = %d, want 1",
			broadcastCount,
		)
	}
}

func TestMinerTokenBridgeOptionsV1LaterValuesOverrideEarlier(
	t *testing.T,
) {
	bridge := &MinerTokenBridge{}

	firstTickCount := 0
	secondTickCount := 0

	opts := bridge.Options(
		miner.Options{
			OnTick: func() {
				firstTickCount++
			},
			Interval: time.Second,
			Animate:  true,
		},
		miner.Options{
			OnTick: func() {
				secondTickCount++
			},
			Interval: 2 * time.Second,
			Animate:  false,
		},
	)

	if opts.OnTick == nil {
		t.Fatal("merged OnTick is nil")
	}

	opts.OnTick()

	if firstTickCount != 0 {
		t.Fatalf(
			"earlier OnTick count = %d, want 0",
			firstTickCount,
		)
	}

	if secondTickCount != 1 {
		t.Fatalf(
			"later OnTick count = %d, want 1",
			secondTickCount,
		)
	}

	if opts.Interval != 2*time.Second {
		t.Fatalf(
			"Interval = %v, want %v",
			opts.Interval,
			2*time.Second,
		)
	}

	if opts.Animate {
		t.Fatal(
			"Animate = true, want later option false",
		)
	}
}
