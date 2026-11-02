package node

import (
	"context"
	"strings"
	"testing"
)

func TestNilRuntimeShutdownIsSafeV1(
	t *testing.T,
) {
	var runtime *Runtime

	err := runtime.Shutdown(
		context.Background(),
	)
	if err == nil {
		t.Fatal(
			"nil runtime shutdown was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"runtime is unavailable",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRuntimeShutdownRejectsNilContextV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.Shutdown(nil)
	if err == nil {
		t.Fatal(
			"nil shutdown context was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"context is unavailable",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestRuntimeShutdownReportsUnavailableChainV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	err := runtime.Shutdown(
		context.Background(),
	)
	if err == nil {
		t.Fatal(
			"runtime without blockchain was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"final chain save",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}
