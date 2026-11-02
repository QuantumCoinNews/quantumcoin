package node

import (
	"strings"
	"testing"
)

func TestRuntimeChainReloadSkipsWritableModeV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	reloaded, err :=
		runtime.ReloadChainFromDiskIfChanged()

	if err != nil {
		t.Fatalf(
			"writable runtime reload returned error: %v",
			err,
		)
	}

	if reloaded {
		t.Fatal(
			"writable runtime unexpectedly reloaded chain",
		)
	}
}

func TestRuntimeChainReloadRejectsMissingConfigV1(
	t *testing.T,
) {
	runtime := NewRuntime(
		nil,
		nil)

	runtime.SetAPIReadOnly(true)

	_, err :=
		runtime.ReloadChainFromDiskIfChanged()

	if err == nil {
		t.Fatal(
			"read-only runtime without config was accepted",
		)
	}

	if !strings.Contains(
		err.Error(),
		"config is unavailable",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestNilRuntimeChainReloadIsSafeV1(
	t *testing.T,
) {
	var runtime *Runtime

	_, err :=
		runtime.ReloadChainFromDiskIfChanged()

	if err == nil {
		t.Fatal(
			"nil runtime chain reload was accepted",
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
