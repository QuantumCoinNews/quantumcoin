package node

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRuntimeSaveChainRejectsUnavailableStateV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.SaveChain()
	if err == nil {
		t.Fatal(
			"runtime without blockchain was accepted",
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

func TestRuntimeRunAutosaveRejectsInvalidIntervalV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.RunAutosave(
		context.Background(),
		0,
		nil,
	)
	if err == nil {
		t.Fatal(
			"zero autosave interval was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"interval must be positive",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRuntimeRunAutosaveStopsOnCancelledContextV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	done := make(chan error, 1)

	go func() {
		done <- runtime.RunAutosave(
			ctx,
			time.Hour,
			nil,
		)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf(
				"cancelled autosave returned error: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"cancelled autosave did not stop",
		)
	}
}
