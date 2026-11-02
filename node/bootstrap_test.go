package node

import (
	"strings"
	"testing"
)

func TestBootstrapWithConfigRejectsNilConfigV1(
	t *testing.T,
) {
	_, err := BootstrapWithConfig(
		nil,
		false,
	)
	if err == nil {
		t.Fatal("nil config was accepted")
	}

	if !strings.Contains(
		err.Error(),
		"config cannot be nil",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestDefaultBootstrapOptionsV1(
	t *testing.T,
) {
	options := DefaultBootstrapOptions()

	if options.ConfigPath != "config.json" {
		t.Fatalf(
			"default config path = %q, want config.json",
			options.ConfigPath,
		)
	}

	if options.StrictChain {
		t.Fatal(
			"default bootstrap unexpectedly enables strict-chain validation",
		)
	}
}
